package dashboard

import (
	"embed"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/vansdevcode/worktree-manager/internal/process"
	"github.com/vansdevcode/worktree-manager/internal/routing"
)

//go:embed index.html
var indexHTML embed.FS

// SiteStatus represents a site with health information.
type SiteStatus struct {
	Domain   string            `json:"domain"`
	Upstream string            `json:"upstream"`
	Meta     map[string]string `json:"meta"`
	Healthy  bool              `json:"healthy"`
}

// StatusResponse is the JSON API response.
type StatusResponse struct {
	Sites []SiteStatus `json:"sites"`
}

// ProcessLister provides process status information.
type ProcessLister interface {
	List() []process.GroupStatus
}

// Server serves the dashboard HTML page and JSON API.
type Server struct {
	routesPath    string
	httpServer    *http.Server
	listener      net.Listener
	processLister ProcessLister
}

// New creates a dashboard server that reads routes from the given path.
func New(routesPath string) *Server {
	s := &Server{routesPath: routesPath}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/processes", s.handleProcesses)
	mux.Handle("/", http.FileServer(http.FS(indexHTML)))

	s.httpServer = &http.Server{Handler: mux}
	return s
}

// SetProcessLister sets the process lister for the /api/processes endpoint.
func (s *Server) SetProcessLister(pl ProcessLister) {
	s.processLister = pl
}

// Start begins listening on a random port and serves in the background.
// Returns the port the server is listening on.
func (s *Server) Start() (int, error) {
	var err error
	s.listener, err = net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}

	go func() { _ = s.httpServer.Serve(s.listener) }()

	return s.listener.Addr().(*net.TCPAddr).Port, nil
}

// Stop gracefully shuts down the dashboard server.
func (s *Server) Stop() error {
	return s.httpServer.Close()
}

func (s *Server) handleStatus(w http.ResponseWriter, _ *http.Request) {
	table, err := routing.Load(s.routesPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sites := table.List()
	resp := StatusResponse{Sites: make([]SiteStatus, 0, len(sites))}
	for _, site := range sites {
		resp.Sites = append(resp.Sites, SiteStatus{
			Domain:   site.Domain,
			Upstream: site.Upstream,
			Meta:     site.Meta,
			Healthy:  checkHealth(site.Upstream),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleProcesses(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.processLister == nil {
		_ = json.NewEncoder(w).Encode([]struct{}{})
		return
	}
	_ = json.NewEncoder(w).Encode(s.processLister.List())
}

// checkHealth performs a TCP connect to the upstream to determine if it's reachable.
func checkHealth(upstream string) bool {
	// upstream is in the form "host:port" (e.g., "localhost:8080")
	if !strings.Contains(upstream, ":") {
		return false
	}
	conn, err := net.DialTimeout("tcp", upstream, 500*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
