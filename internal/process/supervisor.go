package process

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"time"

	"github.com/thejerf/suture/v4"
)

// RouteRegister registers a domain -> upstream route with optional metadata.
type RouteRegister func(domain, upstream string, meta map[string]string) error

// RouteUnregister removes a domain route.
type RouteUnregister func(domain string) error

// Supervisor manages groups of supervised processes loaded from config files.
type Supervisor struct {
	parent     *suture.Supervisor
	logDir     string // base directory for process log files
	mu         sync.Mutex
	groups     map[string]*processGroup // keyed by config absolute path
	register   RouteRegister
	unregister RouteUnregister
}

type processGroup struct {
	supervisor *suture.Supervisor
	token      suture.ServiceToken
	config     *ProjectConfig
	configPath string
	states     map[string]*State
}

// NewSupervisor creates a process supervisor. The register/unregister callbacks
// are used to manage routes when processes have a domain configured.
// logDir is the base directory for per-process log files (e.g. ~/.config/devtree/logs).
func NewSupervisor(parent *suture.Supervisor, logDir string, register RouteRegister, unregister RouteUnregister) *Supervisor {
	return &Supervisor{
		parent:     parent,
		logDir:     logDir,
		groups:     make(map[string]*processGroup),
		register:   register,
		unregister: unregister,
	}
}

// Up loads a devtree.toml config, starts all processes under supervision,
// and registers routes for processes that have a domain.
// workDir is the caller's working directory, used as the default process dir
// when not specified in the config. If empty, defaults to the config file's directory.
func (s *Supervisor) Up(configPath, workDir string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.groups[configPath]; exists {
		return fmt.Errorf("processes from %s are already running, use 'process down' first", configPath)
	}

	cfg, err := LoadConfig(configPath, workDir)
	if err != nil {
		return err
	}

	group := &processGroup{
		config:     cfg,
		configPath: configPath,
		states:     make(map[string]*State),
	}

	group.supervisor = suture.New(cfg.Project.Name, suture.Spec{
		EventHook: func(e suture.Event) {
			log.Printf("[process] %s", e)
		},
		FailureDecay:     30,
		FailureThreshold: 5,
		FailureBackoff:   5 * time.Second,
		Timeout:          10 * time.Second,
	})

	for _, pc := range cfg.Processes {
		pc := pc
		logFile := ""
		if s.logDir != "" {
			logFile = filepath.Join(s.logDir, cfg.Project.Name, pc.Name+".log")
		}
		svc := &Service{
			Config:  pc,
			LogFile: logFile,
			OnUpdate: func(st State) {
				s.mu.Lock()
				defer s.mu.Unlock()
				existing, ok := group.states[st.Name]
				if ok && st.Status == "running" && existing.Status == "crashed" {
					st.Restarts = existing.Restarts + 1
				} else if ok {
					st.Restarts = existing.Restarts
				}
				group.states[st.Name] = &st
			},
		}
		if len(pc.After) > 0 {
			deps := pc.After
			svc.WaitReady = func(ctx context.Context) error {
				return s.waitForDeps(ctx, group, deps)
			}
		}
		group.states[pc.Name] = &State{
			Name:    pc.Name,
			Cmd:     pc.Cmd,
			Port:    pc.Port.Int(),
			Socket:  pc.Socket,
			Domains: pc.Domains,
			Status:  "starting",
			LogFile: logFile,
		}
		group.supervisor.Add(svc)
	}

	group.token = s.parent.Add(group.supervisor)
	s.groups[configPath] = group

	// Register routes for processes with domains.
	for _, pc := range cfg.Processes {
		if len(pc.Domains) == 0 {
			continue
		}
		var upstream string
		switch {
		case pc.Socket != "":
			upstream = "unix/" + pc.Socket
		case pc.Port.Int() != 0:
			upstream = fmt.Sprintf("localhost:%d", pc.Port.Int())
		default:
			continue
		}
		for _, domain := range pc.Domains {
			if err := s.register(domain, upstream, pc.Meta); err != nil {
				log.Printf("[process] warning: failed to register route %s: %v", domain, err)
			}
		}
	}

	return nil
}

// Down stops all processes from a config and unregisters their routes.
func (s *Supervisor) Down(configPath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	group, exists := s.groups[configPath]
	if !exists {
		return fmt.Errorf("no processes running from %s", configPath)
	}

	// Unregister routes.
	for _, pc := range group.config.Processes {
		for _, domain := range pc.Domains {
			if err := s.unregister(domain); err != nil {
				log.Printf("[process] warning: failed to unregister route %s: %v", domain, err)
			}
		}
	}

	// Remove the child supervisor from the parent (stops all processes).
	if err := s.parent.RemoveAndWait(group.token, 10*time.Second); err != nil {
		log.Printf("[process] warning: timeout stopping processes: %v", err)
	}

	delete(s.groups, configPath)
	return nil
}

// List returns the current state of all managed processes.
func (s *Supervisor) List() []GroupStatus {
	s.mu.Lock()
	defer s.mu.Unlock()

	var result []GroupStatus
	for _, group := range s.groups {
		gs := GroupStatus{
			Project:    group.config.Project.Name,
			ConfigPath: group.configPath,
		}
		for _, pc := range group.config.Processes {
			if st, ok := group.states[pc.Name]; ok {
				gs.Processes = append(gs.Processes, *st)
			}
		}
		result = append(result, gs)
	}
	return result
}

// LogFile returns the log file path for a named process, or empty if not found.
func (s *Supervisor) LogFile(name string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, group := range s.groups {
		if st, ok := group.states[name]; ok {
			return st.LogFile
		}
	}
	return ""
}

// Serve implements suture.Service. It blocks until the context is done.
func (s *Supervisor) Serve(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

func (s *Supervisor) String() string {
	return "process-supervisor"
}

// waitForDeps polls until all named dependencies have status "running" or "completed".
func (s *Supervisor) waitForDeps(ctx context.Context, group *processGroup, deps []string) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if s.depsReady(group, deps) {
				return nil
			}
		}
	}
}

func (s *Supervisor) depsReady(group *processGroup, deps []string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, dep := range deps {
		st, ok := group.states[dep]
		if !ok {
			return false
		}
		if st.Status != "running" && st.Status != "completed" {
			return false
		}
	}
	return true
}

// GroupStatus holds the status of all processes in a project.
type GroupStatus struct {
	Project    string  `json:"project"`
	ConfigPath string  `json:"config_path"`
	Processes  []State `json:"processes"`
}
