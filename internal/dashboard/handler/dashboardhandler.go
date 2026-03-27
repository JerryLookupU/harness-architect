package handler

import (
	"net/http"

	"klein-harness/internal/dashboard/logic"
	"klein-harness/internal/dashboard/svc"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func ProjectDashboardHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logic.NewProjectDashboardLogic(r.Context(), svcCtx)
		view, err := l.ProjectDashboard()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, view)
	}
}
