package middleware

import (
	"bytes"
	baseCtx "context"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"

	"github.com/pquerna/cachecontrol/cacheobject"

	"github.com/darkweak/souin/pkg/rfc"
	"github.com/darkweak/storages/core"
)

func (s *SouinBaseHandler) triggerBackgroundRevalidation(
	validator *core.Revalidator,
	req *http.Request,
	next handlerFunc,
	requestCc *cacheobject.RequestCacheDirectives,
	cachedKey string,
	uri string,
) {
	backgroundReq := req.Clone(baseCtx.WithoutCancel(req.Context()))
	backgroundReq.Header = req.Header.Clone()

	go func() {
		recorder := httptest.NewRecorder()
		backgroundWriter := NewCustomWriter(backgroundReq, recorder, new(bytes.Buffer))
		_ = s.Revalidate(validator, next, backgroundWriter, backgroundReq, requestCc, cachedKey, uri)
	}()
}

func (s *SouinBaseHandler) serveStaleWhileRevalidateResponse(
	customWriter *CustomWriter,
	response *http.Response,
	req *http.Request,
	next handlerFunc,
	validator *core.Revalidator,
	requestCc *cacheobject.RequestCacheDirectives,
	cachedKey string,
	uri string,
) error {
	customWriter.WriteHeader(response.StatusCode)
	rfc.HitStaleCache(&response.Header)
	maps.Copy(customWriter.Header(), response.Header)
	customWriter.handleBuffer(func(b *bytes.Buffer) {
		_, _ = io.Copy(b, response.Body)
	})
	_, err := customWriter.Send()
	s.triggerBackgroundRevalidation(validator, req, next, requestCc, cachedKey, uri)

	return err
}
