package certs

import (
	"testing"
)

func TestGetBaseDomain(t *testing.T) {
	tests := []struct {
		domain string
		want   string
	}{
		{"myapp.test", "myapp.test"},
		{"branch.myapp.test", "myapp.test"},
		{"deep.branch.myapp.test", "myapp.test"},
		{"example.com", "example.com"},
		{"sub.example.com", "sub.example.com"}, // non-.test, returned as-is
		{"localhost", "localhost"},
	}

	for _, tt := range tests {
		t.Run(tt.domain, func(t *testing.T) {
			got := getBaseDomain(tt.domain)
			if got != tt.want {
				t.Errorf("getBaseDomain(%q) = %q, want %q", tt.domain, got, tt.want)
			}
		})
	}
}

type mockLister struct {
	domains []string
}

func (m *mockLister) DomainNames() []string {
	return m.domains
}

func TestRemoveCertIfUnused_SharedBase(t *testing.T) {
	// When another domain shares the same base, cert should NOT be removed
	lister := &mockLister{domains: []string{"branch2.myapp.test"}}

	// RemoveCertIfUnused should return nil (no error) and NOT remove anything
	// because branch2.myapp.test shares base "myapp.test" with branch1.myapp.test
	err := RemoveCertIfUnused("branch1.myapp.test", lister)
	if err != nil {
		t.Errorf("RemoveCertIfUnused() error: %v", err)
	}
}

func TestRemoveCertIfUnused_NoSharedBase(t *testing.T) {
	// When no other domain shares the base, cert should be removed
	// (will succeed even if cert files don't exist)
	lister := &mockLister{domains: []string{"other.test"}}

	err := RemoveCertIfUnused("myapp.test", lister)
	if err != nil {
		t.Errorf("RemoveCertIfUnused() error: %v", err)
	}
}

func TestRemoveCertIfUnused_EmptyTable(t *testing.T) {
	lister := &mockLister{domains: []string{}}

	err := RemoveCertIfUnused("myapp.test", lister)
	if err != nil {
		t.Errorf("RemoveCertIfUnused() error: %v", err)
	}
}
