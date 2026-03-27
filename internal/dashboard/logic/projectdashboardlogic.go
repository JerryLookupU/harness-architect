package logic

import (
	"context"

	"klein-harness/internal/dashboard/svc"
	"klein-harness/internal/query"

	"github.com/zeromicro/go-zero/core/logx"
)

type ProjectDashboardLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewProjectDashboardLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ProjectDashboardLogic {
	return &ProjectDashboardLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ProjectDashboardLogic) ProjectDashboard() (query.Dashboard, error) {
	return query.ProjectDashboard(l.svcCtx.Root)
}
