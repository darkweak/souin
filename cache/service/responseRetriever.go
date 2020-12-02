package service

import (
	"github.com/darkweak/souin/cache/types"
	"net/http"
	"strings"
)

// ServeResponse serve the response
func ServeResponse(
	res http.ResponseWriter,
	req *http.Request,
	retriever types.RetrieverResponsePropertiesInterface,
	callback func(rw http.ResponseWriter, rq *http.Request, r types.RetrieverResponsePropertiesInterface, key string),
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
		path+headers,
	)
}
