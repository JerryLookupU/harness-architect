package dashboard

import (
	"embed"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"strconv"
	"strings"

	"klein-harness/internal/dashboard/handler"
	"klein-harness/internal/dashboard/svc"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/rest"
)

//go:embed web/dist
var assets embed.FS

// dashboard config
type Config struct {
	rest.RestConf
	Root string
}

func NewConfig(root, addr string) (Config, error) {
	host, portText, err := net.SplitHostPort(strings.TrimSpace(addr))
	if err != nil {
		return Config{}, fmt.Errorf("invalid dashboard addr %q: %w", addr, err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		return Config{}, fmt.Errorf("invalid dashboard port %q: %w", portText, err)
	}
	return Config{
		Root: root,
		RestConf: rest.RestConf{
			ServiceConf: service.ServiceConf{
				Name: "harness-dashboard",
				Mode: "dev",
				Log: logx.LogConf{
					ServiceName: "harness-dashboard",
					Mode:        "console",
					Encoding:    "plain",
				},
			},
			Host:     host,
			Port:     port,
			Timeout:  30000,
			MaxConns: 10000,
			MaxBytes: 8 << 20,
		},
	}, nil
}

func NewServer(config Config) (*rest.Server, error) {
	dist, err := fs.Sub(assets, "web/dist")
	if err != nil {
		return nil, err
	}
	svcCtx := svc.NewServiceContext(config.Root)
	server, err := rest.NewServer(config.RestConf,
		rest.WithFileServer("/", http.FS(dist)),
		rest.WithNotFoundHandler(handler.NewSPAFallbackHandler(dist)),
	)
	if err != nil {
		return nil, err
	}
	server.Use(noCacheMiddleware())
	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/api/dashboard",
		Handler: handler.ProjectDashboardHandler(svcCtx),
	})
	return server, nil
}

func noCacheMiddleware() rest.Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store")
			if strings.HasPrefix(r.URL.Path, "/api/") {
				w.Header().Set("X-Content-Type-Options", "nosniff")
			}
			next(w, r)
		}
	}
}
