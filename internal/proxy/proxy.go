package proxy

import (
	"encoding/json"
	"fmt"

	"github.com/caddyserver/caddy/v2"
	_ "github.com/caddyserver/caddy/v2/modules/standard"

	"github.com/vansdevcode/worktree-manager/internal/routing"
)

// Server wraps an embedded Caddy instance for TLS-terminating reverse proxying.
type Server struct {
	httpPort      int
	httpsPort     int
	dashboardPort int
}

// New creates a proxy server that will listen on the given ports.
func New(httpPort, httpsPort int) *Server {
	return &Server{
		httpPort:  httpPort,
		httpsPort: httpsPort,
	}
}

// SetDashboardPort configures the internal port for the dashboard reverse proxy route.
func (s *Server) SetDashboardPort(port int) {
	s.dashboardPort = port
}

// Start loads the Caddy config built from the routing table.
func (s *Server) Start(table *routing.Table) error {
	return s.load(table)
}

// Reload rebuilds and hot-reloads the Caddy config from the current routing table.
func (s *Server) Reload(table *routing.Table) error {
	return s.load(table)
}

// Stop gracefully stops the Caddy instance.
func (s *Server) Stop() error {
	return caddy.Stop()
}

func (s *Server) load(table *routing.Table) error {
	cfg := s.buildConfig(table)

	data, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling caddy config: %w", err)
	}

	if err := caddy.Load(data, true); err != nil {
		return fmt.Errorf("loading caddy config: %w", err)
	}

	return nil
}

// buildConfig creates a Caddy JSON config from the routing table.
func (s *Server) buildConfig(table *routing.Table) map[string]any {
	routes := make([]map[string]any, 0, len(table.Sites)+1)
	subjects := make([]string, 0, len(table.Sites)+1)

	// Add dashboard route if configured.
	if s.dashboardPort > 0 {
		subjects = append(subjects, "dashboard.devtree.test")
		routes = append(routes, map[string]any{
			"match": []map[string]any{
				{"host": []string{"dashboard.devtree.test"}},
			},
			"handle": []map[string]any{
				{
					"handler": "reverse_proxy",
					"upstreams": []map[string]any{
						{"dial": fmt.Sprintf("127.0.0.1:%d", s.dashboardPort)},
					},
				},
			},
		})
	}

	for domain, site := range table.Sites {
		subjects = append(subjects, domain)
		route := map[string]any{
			"match": []map[string]any{
				{"host": []string{domain}},
			},
			"handle": []map[string]any{
				{
					"handler": "reverse_proxy",
					"upstreams": []map[string]any{
						{"dial": site.Upstream},
					},
				},
			},
		}
		routes = append(routes, route)
	}

	cfg := map[string]any{
		"apps": map[string]any{
			"http": map[string]any{
				"http_port":  s.httpPort,
				"https_port": s.httpsPort,
				"servers": map[string]any{
					"srv0": map[string]any{
						"listen": []string{fmt.Sprintf(":%d", s.httpsPort)},
						"routes": routes,
						"tls_connection_policies": []map[string]any{{}},
					},
				},
			},
			"tls": map[string]any{
				"automation": map[string]any{
					"policies": []map[string]any{
						{
							"subjects": subjects,
							"issuers": []map[string]any{
								{"module": "internal"},
							},
						},
					},
				},
			},
		},
	}

	return cfg
}
