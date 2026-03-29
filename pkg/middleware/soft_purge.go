package middleware

import (
	"bytes"
	baseCtx "context"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"

	"github.com/darkweak/souin/pkg/api"
	"github.com/darkweak/souin/pkg/api/prometheus"
	"github.com/darkweak/souin/pkg/rfc"
	"github.com/darkweak/souin/pkg/storage/types"
	"github.com/pquerna/cachecontrol/cacheobject"
)

func (s *SouinBaseHandler) clearSoftPurgeMarker(storer types.Storer, storedKey string) {
	if storedKey == "" {
		return
	}

	storer.Delete(api.SoftPurgeMarkerKey(storedKey))
}

func (s *SouinBaseHandler) isSoftPurgedResponse(storer types.Storer, response *http.Response) bool {
	if storer == nil || response == nil {
		return false
	}

	storedKey := response.Header.Get(rfc.StoredKeyHeader)
	if storedKey == "" {
		return false
	}

	return len(storer.Get(api.SoftPurgeMarkerKey(storedKey))) > 0
}

func hasSoftPurgeValidators(response *http.Response) bool {
	if response == nil {
		return false
	}

	return response.Header.Get("Etag") != "" || response.Header.Get("Last-Modified") != ""
}

func (s *SouinBaseHandler) getSoftPurgeDetail(
	response *http.Response,
	requestCc *cacheobject.RequestCacheDirectives,
) (string, bool) {
	if response == nil {
		return "SOFT-PURGED", false
	}

	if hasSoftPurgeValidators(response) {
		return "SOFT-PURGE-REVALIDATE", true
	}

	responseCc, _ := cacheobject.ParseResponseCacheControl(rfc.HeaderAllCommaSepValuesString(response.Header, "Cache-Control"))
	if responseCc != nil && responseCc.StaleWhileRevalidate > 0 {
		return "SOFT-PURGE-SWR", true
	}

	if (responseCc != nil && responseCc.StaleIfError > -1) || requestCc.StaleIfError > 0 {
		return "SOFT-PURGE-SIE", false
	}

	return "SOFT-PURGED", false
}

func cloneBodyForSoftPurge(response *http.Response) []byte {
	if response == nil || response.Body == nil {
		return nil
	}

	body, _ := io.ReadAll(response.Body)
	response.Body = io.NopCloser(bytes.NewReader(body))

	return body
}

func mergeRevalidatedHeaders(staleHeaders, revalidatedHeaders http.Header) http.Header {
	merged := staleHeaders.Clone()
	maps.Copy(merged, revalidatedHeaders)

	return merged
}

func (s *SouinBaseHandler) storeRevalidatedStaleResponse(
	req *http.Request,
	requestCc *cacheobject.RequestCacheDirectives,
	cachedKey string,
	uri string,
	statusCode int,
	headers http.Header,
	body []byte,
) error {
	recorder := httptest.NewRecorder()
	customWriter := NewCustomWriter(req, recorder, new(bytes.Buffer))
	maps.Copy(customWriter.Header(), headers)
	customWriter.WriteHeader(statusCode)
	_, _ = customWriter.Write(body)

	return s.Store(customWriter, req, requestCc, cachedKey, uri)
}

func (s *SouinBaseHandler) triggerSoftPurgeBackgroundRefresh(
	storedKey string,
	req *http.Request,
	next handlerFunc,
	requestCc *cacheobject.RequestCacheDirectives,
	cachedKey string,
	uri string,
	response *http.Response,
	body []byte,
) {
	if storedKey == "" {
		return
	}

	if _, loaded := s.backgroundRefreshes.LoadOrStore(storedKey, struct{}{}); loaded {
		s.Configuration.GetLogger().Infof("Skipping duplicate background refresh for soft-purged key %s", storedKey)
		prometheus.Increment(prometheus.SoftPurgeRefreshDeduped)
		return
	}

	prometheus.Increment(prometheus.SoftPurgeRefreshCounter)
	backgroundReq := req.Clone(baseCtx.WithoutCancel(req.Context()))
	backgroundReq.Header = req.Header.Clone()
	backgroundReq.Header.Del("Cache-Control")

	if etag := response.Header.Get("Etag"); etag != "" {
		backgroundReq.Header.Set("If-None-Match", etag)
	}
	if lastModified := response.Header.Get("Last-Modified"); lastModified != "" {
		backgroundReq.Header.Set("If-Modified-Since", lastModified)
	}

	go func() {
		defer s.backgroundRefreshes.Delete(storedKey)

		s.Configuration.GetLogger().Infof("Starting background refresh for soft-purged key %s", storedKey)
		recorder := httptest.NewRecorder()
		backgroundWriter := NewCustomWriter(backgroundReq, recorder, new(bytes.Buffer))

		refreshErr := s.Upstream(backgroundWriter, backgroundReq, next, requestCc, cachedKey, uri, false)
		statusCode := backgroundWriter.GetStatusCode()

		if refreshErr != nil {
			s.Configuration.GetLogger().Warnf("Background refresh failed for soft-purged key %s: %v", storedKey, refreshErr)
			prometheus.Increment(prometheus.SoftPurgeRefreshFailure)
			return
		}

		if statusCode == http.StatusNotModified {
			mergedHeaders := mergeRevalidatedHeaders(response.Header, backgroundWriter.Header())
			if err := s.storeRevalidatedStaleResponse(backgroundReq, requestCc, cachedKey, uri, response.StatusCode, mergedHeaders, body); err != nil {
				s.Configuration.GetLogger().Warnf("Background 304 revalidation failed for soft-purged key %s: %v", storedKey, err)
				prometheus.Increment(prometheus.SoftPurgeRefreshFailure)
				return
			}

			s.Configuration.GetLogger().Infof("Background refresh revalidated soft-purged key %s with 304", storedKey)
			prometheus.Increment(prometheus.SoftPurgeRefreshSuccess)
			return
		}

		s.Configuration.GetLogger().Infof("Background refresh completed for soft-purged key %s with status %d", storedKey, statusCode)
		prometheus.Increment(prometheus.SoftPurgeRefreshSuccess)
	}()
}

func (s *SouinBaseHandler) serveSoftPurgedResponse(
	customWriter *CustomWriter,
	response *http.Response,
	storerName string,
	req *http.Request,
	next handlerFunc,
	requestCc *cacheobject.RequestCacheDirectives,
	cachedKey string,
	uri string,
) error {
	storedKey := response.Header.Get(rfc.StoredKeyHeader)
	body := cloneBodyForSoftPurge(response)
	detail, shouldRefresh := s.getSoftPurgeDetail(response, requestCc)
	rfc.SetCacheStatusHeader(response, storerName)
	rfc.HitStaleCache(&response.Header)
	response.Header.Set("Cache-Status", response.Header.Get("Cache-Status")+"; detail="+detail)
	maps.Copy(customWriter.Header(), response.Header)
	customWriter.WriteHeader(response.StatusCode)
	customWriter.handleBuffer(func(b *bytes.Buffer) {
		_, _ = b.Write(body)
	})

	_, err := customWriter.Send()
	prometheus.Increment(prometheus.CachedResponseCounter)
	prometheus.Increment(prometheus.SoftPurgeHitCounter)
	if err != nil {
		return err
	}

	if shouldRefresh {
		s.triggerSoftPurgeBackgroundRefresh(storedKey, req, next, requestCc, cachedKey, uri, response, body)
	} else {
		s.Configuration.GetLogger().Infof("Soft-purged key %s served stale without background refresh because no validators or stale-while-revalidate directives were present", storedKey)
	}

	return nil
}
