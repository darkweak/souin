package handler

import (
	"net/http"

	"github.com/darkweak/souin/plugins/go-zero/examples/internal/logic"
	"github.com/darkweak/souin/plugins/go-zero/examples/internal/svc"
	"github.com/darkweak/souin/plugins/go-zero/examples/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func mainHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.CacheReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.Error(w, err)
			return
		}

		l := logic.NewMainLogic(r.Context(), svcCtx)
		resp, err := l.Main(&req)
		if err != nil {
			httpx.Error(w, err)
		} else {
			httpx.OkJson(w, resp)
		}
	}
}
