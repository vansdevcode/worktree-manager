package process

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

// ProjectConfig represents a devtree project configuration (devtree.toml or devtree.json).
type ProjectConfig struct {
	Project   ProjectMeta     `toml:"project" json:"project"`
	Processes []ProcessConfig `toml:"process" json:"process"`
}

// ProjectMeta holds project-level metadata.
type ProjectMeta struct {
	Name   string `toml:"name" json:"name"`
	Domain string `toml:"domain" json:"domain,omitempty"`
}

// ProcessConfig defines a single supervised process.
type ProcessConfig struct {
	Name    string            `toml:"name" json:"name"`
	Cmd     string            `toml:"cmd" json:"cmd"`
	Port    EnvInt            `toml:"port" json:"port,omitempty"`
	Socket  string            `toml:"socket" json:"socket,omitempty"`
	Dir     string            `toml:"dir" json:"dir,omitempty"`
	Env     map[string]string `toml:"env" json:"env,omitempty"`
	Domains Domains           `toml:"domain" json:"domain,omitempty"`
	Meta    map[string]string `toml:"meta" json:"meta,omitempty"`
	Restart string            `toml:"restart" json:"restart,omitempty"` // "always" (default), "on_failure", "never"
	After   []string          `toml:"after" json:"after,omitempty"`    // process names that must be running first
}

// Domains is a list of domain names that can be unmarshaled from either a
// single string or an array of strings in both TOML and JSON.
//
//	domain = "myapp.test"
//	domain = ["myapp.test", "admin.myapp.test"]
type Domains []string

// UnmarshalTOML handles both a TOML string and a TOML array of strings.
func (d *Domains) UnmarshalTOML(v any) error {
	switch val := v.(type) {
	case string:
		*d = Domains{val}
	case []any:
		for _, elem := range val {
			s, ok := elem.(string)
			if !ok {
				return fmt.Errorf("domain array elements must be strings, got %T", elem)
			}
			*d = append(*d, s)
		}
	default:
		return fmt.Errorf("domain must be a string or array of strings, got %T", v)
	}
	return nil
}

// UnmarshalJSON handles both a JSON string and a JSON array of strings.
func (d *Domains) UnmarshalJSON(data []byte) error {
	// Try string first.
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*d = Domains{s}
		return nil
	}
	// Try array.
	var arr []string
	if err := json.Unmarshal(data, &arr); err != nil {
		return fmt.Errorf("domain must be a string or array of strings, got %s", string(data))
	}
	*d = arr
	return nil
}

// EnvInt is an integer that can be specified as a number or a string with
// environment variable references (e.g. "$VITE_PORT"). The env vars are
// expanded and parsed to int during config resolution.
type EnvInt struct {
	Raw      string // original string form (set when parsed from a string)
	Resolved int    // final integer value after env expansion
}

// Int returns the resolved integer value.
func (e EnvInt) Int() int { return e.Resolved }

// UnmarshalTOML implements toml.Unmarshaler to handle both TOML integers and strings.
func (e *EnvInt) UnmarshalTOML(v any) error {
	switch val := v.(type) {
	case int64:
		e.Resolved = int(val)
	case string:
		e.Raw = val
	default:
		return fmt.Errorf("port must be a number or string, got %T", v)
	}
	return nil
}

// UnmarshalJSON handles both JSON numbers and strings.
func (e *EnvInt) UnmarshalJSON(data []byte) error {
	// Try number first.
	var n int
	if err := json.Unmarshal(data, &n); err == nil {
		e.Resolved = n
		return nil
	}
	// Try string.
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("port must be a number or string, got %s", string(data))
	}
	e.Raw = s
	return nil
}

// MarshalJSON writes the resolved integer.
func (e EnvInt) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.Resolved)
}

// resolve expands env vars in Raw and sets Resolved. If Raw is empty,
// Resolved is already set (came from a TOML integer or JSON number).
// The mapping function is used to look up variable values; if nil, os.Getenv is used.
func (e *EnvInt) resolve(mapping func(string) string) error {
	if e.Raw == "" {
		return nil
	}
	if mapping == nil {
		mapping = os.Getenv
	}
	expanded := os.Expand(e.Raw, mapping)
	if expanded == "" {
		e.Resolved = 0
		return nil
	}
	n, err := strconv.Atoi(expanded)
	if err != nil {
		return fmt.Errorf("port %q (expanded from %q) is not a valid integer", expanded, e.Raw)
	}
	e.Resolved = n
	return nil
}

// LoadConfig reads and parses a devtree config file (TOML or JSON), resolving defaults.
// The format is determined by file extension: .json for JSON, anything else for TOML.
// workDir overrides the base directory used for resolving relative paths and as the
// default process working directory. If empty, defaults to the config file's parent directory.
func LoadConfig(path string, workDir ...string) (*ProjectConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg ProjectConfig
	switch {
	case strings.HasSuffix(path, ".json"):
		err = json.Unmarshal(data, &cfg)
	default:
		err = toml.Unmarshal(data, &cfg)
	}
	if err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	baseDir := filepath.Dir(path)
	if len(workDir) > 0 && workDir[0] != "" {
		baseDir = workDir[0]
	}

	if err := cfg.resolve(baseDir); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// resolve applies defaults and validates the config.
// baseDir is used as the default process working directory and for resolving relative dir values.
func (c *ProjectConfig) resolve(baseDir string) error {
	if c.Project.Name == "" {
		return fmt.Errorf("project.name is required")
	}

	if c.Project.Domain == "" {
		c.Project.Domain = c.Project.Name + ".test"
	}

	if len(c.Processes) == 0 {
		return fmt.Errorf("at least one [[process]] is required")
	}

	seen := make(map[string]bool)
	for i := range c.Processes {
		p := &c.Processes[i]
		if p.Name == "" {
			return fmt.Errorf("process[%d]: name is required", i)
		}
		if seen[p.Name] {
			return fmt.Errorf("process[%d]: duplicate name %q", i, p.Name)
		}
		seen[p.Name] = true

		if p.Cmd == "" {
			return fmt.Errorf("process %q: cmd is required", p.Name)
		}
		// Build a lookup that checks the process's own env first, then OS env.
		// This allows port = "$VITE_PORT" when VITE_PORT is defined in [process.env].
		envLookup := envMapper(p.Env)
		if err := p.Port.resolve(envLookup); err != nil {
			return fmt.Errorf("process %q: %w", p.Name, err)
		}
		if p.Port.Int() != 0 && p.Socket != "" {
			return fmt.Errorf("process %q: port and socket are mutually exclusive", p.Name)
		}
		if len(p.Domains) > 0 && p.Port.Int() == 0 && p.Socket == "" {
			return fmt.Errorf("process %q: port or socket is required when domain is set", p.Name)
		}
		if p.Dir == "" {
			p.Dir = baseDir
		} else if !filepath.IsAbs(p.Dir) {
			p.Dir = filepath.Join(baseDir, p.Dir)
		}

		switch p.Restart {
		case "", "always", "on_failure", "never":
			// valid
		default:
			return fmt.Errorf("process %q: invalid restart policy %q (must be always, on_failure, or never)", p.Name, p.Restart)
		}
	}

	// Validate after references.
	for i := range c.Processes {
		p := &c.Processes[i]
		for _, dep := range p.After {
			if !seen[dep] {
				return fmt.Errorf("process %q: after references unknown process %q", p.Name, dep)
			}
			if dep == p.Name {
				return fmt.Errorf("process %q: cannot depend on itself", p.Name)
			}
		}
	}

	return nil
}

// envMapper returns a mapping function that checks the process env map first,
// then falls back to os.Getenv. Values in the env map are themselves expanded
// against os.Getenv. Returns nil if the map is empty (callers fall back to os.Getenv).
func envMapper(env map[string]string) func(string) string {
	if len(env) == 0 {
		return nil
	}
	return func(key string) string {
		if v, ok := env[key]; ok {
			return os.ExpandEnv(v)
		}
		return os.Getenv(key)
	}
}

// configFiles lists the filenames to search for, in priority order.
var configFiles = []string{"devtree.toml", "devtree.json"}

// FindConfig walks up from dir looking for devtree.toml or devtree.json.
// Prefers devtree.toml if both exist in the same directory.
// Returns the absolute path if found, or an error.
func FindConfig(dir string) (string, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}

	for {
		for _, name := range configFiles {
			path := filepath.Join(dir, name)
			if _, err := os.Stat(path); err == nil {
				return path, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("devtree.toml or devtree.json not found")
		}
		dir = parent
	}
}
