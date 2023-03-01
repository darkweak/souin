package main

/*
func Benchmark_Souin_Handler(b *testing.B) {
	c := configuration.GetConfiguration()
	rc := coalescing.Initialize()
	retriever := souinPluginInitializerFromConfiguration(c)
	for i := 0; i < b.N; i++ {
		writer := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "http://domain.com/"+strconv.Itoa(i), nil)
		request.Header.Set("Date", time.Now().UTC().Format(time.RFC1123))
		request = retriever.GetContext().Method.SetContext(request)

		if !plugins.CanHandle(request, retriever) {
			writer.Header().Set("Cache-Status", "Souin; fwd=uri-miss")
			return
		}

		request = retriever.GetContext().SetContext(request)
		callback := func(rw http.ResponseWriter, rq *http.Request, ret souintypes.SouinRetrieverResponseProperties) error {
			rr := service.RequestReverseProxy(rq, ret)
			select {
			case <-rq.Context().Done():
				c.GetLogger().Debug("The request was canceled by the user.")
				return &errors.CanceledRequestContextError{}
			default:
				rr.Proxy.ServeHTTP(rw, rq)
			}

			return nil
		}
		if plugins.HasMutation(request, writer) {
			_ = callback(writer, request, *retriever)
		}
		retriever.SetMatchedURLFromRequest(request)
		coalescing.ServeResponse(writer, request, retriever, plugins.DefaultSouinPluginCallback, rc, func(_ http.ResponseWriter, _ *http.Request) error {
			return callback(writer, request, *retriever)
		})
	}
}
*/
