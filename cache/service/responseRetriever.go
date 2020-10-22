package service

import (
	"net/http"
	"github.com/darkweak/souin/cache/types"
	"net/url"
	"strings"
	"encoding/json"
)

func callback(
	res http.ResponseWriter,
	req *http.Request,
	retriever types.RetrieverResponsePropertiesInterface,
	key string,
) {

	u, _ := url.Parse(retriever.GetConfiguration().GetReverseProxyURL())
	ctx := req.Context()
	responses := make(chan types.ReverseResponse)

	alreadyHaveResponse := false
	alreadySent := false

	go func() {
		if http.MethodGet == req.Method {
			if !alreadyHaveResponse {
				r := retriever.GetProvider().GetRequestInCache(key)
				responses <- retriever.GetProvider().GetRequestInCache(key)
				if "" != r.Response {
					alreadyHaveResponse = true
				}
			}
		}
		if !alreadyHaveResponse || http.MethodGet != req.Method {
			responses <- RequestReverseProxy(req, u, retriever)
		}
	}()

	if http.MethodGet == req.Method {
		response, open := <-responses
		if open && http.MethodGet == req.Method && "" != response.Response {
			close(responses)
			var responseJSON types.RequestResponse
			err := json.Unmarshal([]byte(response.Response), &responseJSON)
			if err != nil {
				panic(err)
			}
			for k, v := range responseJSON.Headers {
				res.Header().Set(k, v[0])
			}
			alreadySent = true
			res.Write(responseJSON.Body)
		}
	}

	if !alreadySent {
		req = req.WithContext(ctx)
		response2 := <-responses
		close(responses)
		response2.Proxy.ServeHTTP(res, req)
	}
}

func ServeResponse(
	res http.ResponseWriter,
	req *http.Request,
	retriever types.RetrieverResponsePropertiesInterface,
) {
	path := req.Host + req.URL.Path
	regexpURL := retriever.GetRegexpUrls().FindString(path)
	if "" != regexpURL {
		retriever.SetMatchedURL(retriever.GetConfiguration().GetUrls()[regexpURL])
	}
	headers := ""
	if retriever.GetMatchedURL().Headers != nil && len(retriever.GetMatchedURL().Headers) > 0 {
		for _, h := range retriever.GetMatchedURL().Headers {
			headers += strings.ReplaceAll(req.Header.Get(h), " ", "")
		}
	}

	callback(
		res,
		req,
		retriever,
		string(path + headers),
	)
}
