package routing

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// Site represents a registered route entry.
type Site struct {
	Domain   string            `yaml:"-"`
	Upstream string            `yaml:"upstream"`
	Meta     map[string]string `yaml:"meta,omitempty"`
}

// Table holds the routing state.
type Table struct {
	Sites map[string]*Site `yaml:"sites"`
}

// NewTable creates an empty routing table.
func NewTable() *Table {
	return &Table{Sites: make(map[string]*Site)}
}

// Register adds or updates a route in the table.
func (t *Table) Register(domain, upstream string, meta map[string]string) {
	t.Sites[domain] = &Site{
		Domain:   domain,
		Upstream: upstream,
		Meta:     meta,
	}
}

// Unregister removes a route from the table. Returns an error if the domain doesn't exist.
func (t *Table) Unregister(domain string) error {
	if _, ok := t.Sites[domain]; !ok {
		return fmt.Errorf("domain %q is not registered", domain)
	}
	delete(t.Sites, domain)
	return nil
}

// Get returns a site by domain and whether it exists.
func (t *Table) Get(domain string) (*Site, bool) {
	s, ok := t.Sites[domain]
	if ok {
		s.Domain = domain
	}
	return s, ok
}

// List returns all sites sorted alphabetically by domain.
func (t *Table) List() []Site {
	result := make([]Site, 0, len(t.Sites))
	for domain, site := range t.Sites {
		s := *site
		s.Domain = domain
		result = append(result, s)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Domain < result[j].Domain
	})
	return result
}

// Load reads and parses a routing table from a YAML file.
// Returns an empty table if the file doesn't exist.
func Load(path string) (*Table, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewTable(), nil
		}
		return nil, fmt.Errorf("reading routes file: %w", err)
	}

	t := NewTable()
	if err := yaml.Unmarshal(data, t); err != nil {
		return nil, fmt.Errorf("parsing routes file: %w", err)
	}

	if t.Sites == nil {
		t.Sites = make(map[string]*Site)
	}

	return t, nil
}

// Save writes the routing table to a YAML file atomically (temp file + rename).
func Save(path string, table *Table) error {
	data, err := yaml.Marshal(table)
	if err != nil {
		return fmt.Errorf("marshaling routes: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".routes-*.yaml")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("writing temp file: %w", err)
	}

	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("closing temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("renaming temp file: %w", err)
	}

	return nil
}
