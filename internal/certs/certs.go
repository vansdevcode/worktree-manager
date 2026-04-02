package certs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jittering/truststore"
)

// DefaultCertsDir returns the default directory for storing generated certificates.
func DefaultCertsDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "devtree", "certs")
}

// DefaultCADir returns the default directory for the local CA.
func DefaultCADir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "devtree", "ca")
}

// setCAROOT ensures truststore uses our CA directory.
func setCAROOT() {
	_ = os.Setenv("CAROOT", DefaultCADir())
}

// newLib initializes truststore with our CA directory.
func newLib() (*truststore.MkcertLib, error) {
	setCAROOT()
	return truststore.NewLib()
}

// EnsureCA creates the local CA if it doesn't exist and installs it into
// system trust stores. This is idempotent — safe to call multiple times.
func EnsureCA() error {
	caDir := DefaultCADir()
	if err := os.MkdirAll(caDir, 0755); err != nil {
		return fmt.Errorf("creating CA directory: %w", err)
	}

	ml, err := newLib()
	if err != nil {
		return fmt.Errorf("initializing truststore: %w", err)
	}

	if err := ml.Install(); err != nil {
		return fmt.Errorf("installing CA: %w", err)
	}

	return nil
}

// EnsureCert generates a TLS certificate for the given domain if one doesn't
// already exist. For .test domains, a wildcard certificate (*.domain) is also
// generated to cover branch subdomains.
// Returns the cert and key file paths.
func EnsureCert(domain string) (certFile, keyFile string, err error) {
	certsDir := DefaultCertsDir()
	if err := os.MkdirAll(certsDir, 0755); err != nil {
		return "", "", fmt.Errorf("creating certs directory: %w", err)
	}

	baseDomain := getBaseDomain(domain)
	hosts := []string{baseDomain, "*." + baseDomain}

	ml, err := newLib()
	if err != nil {
		return "", "", fmt.Errorf("initializing truststore: %w", err)
	}

	// Check if cert already exists
	cert, err := ml.CertFile(hosts, certsDir)
	if err != nil {
		return "", "", fmt.Errorf("computing cert path: %w", err)
	}

	if cert.Exists() {
		return cert.CertFile, cert.KeyFile, nil
	}

	cert, err = ml.MakeCert(hosts, certsDir)
	if err != nil {
		return "", "", fmt.Errorf("generating certificate for %s: %w", baseDomain, err)
	}

	return cert.CertFile, cert.KeyFile, nil
}

// FindCert returns the cert and key file paths for a domain. It checks for
// a wildcard cert covering the domain's base. Returns empty strings if no cert exists.
func FindCert(domain string) (certFile, keyFile string) {
	certsDir := DefaultCertsDir()
	baseDomain := getBaseDomain(domain)
	hosts := []string{baseDomain, "*." + baseDomain}

	ml, err := newLib()
	if err != nil {
		return "", ""
	}

	cert, err := ml.CertFile(hosts, certsDir)
	if err != nil {
		return "", ""
	}

	if cert.Exists() {
		return cert.CertFile, cert.KeyFile
	}

	return "", ""
}

// DomainLister is an interface for checking remaining registered domains.
type DomainLister interface {
	// Sites returns a map of domain -> site data. Used to check if
	// other domains share the same base domain cert.
	DomainNames() []string
}

// RemoveCertIfUnused deletes certificate files for a domain's base domain,
// but only if no other registered domains in the table share the same base.
func RemoveCertIfUnused(domain string, lister DomainLister) error {
	baseDomain := getBaseDomain(domain)

	// Check if any remaining domain shares the same base
	for _, d := range lister.DomainNames() {
		if d != domain && getBaseDomain(d) == baseDomain {
			return nil // cert still in use
		}
	}

	return RemoveCert(domain)
}

// RemoveCert deletes certificate files for a domain's base domain.
func RemoveCert(domain string) error {
	certsDir := DefaultCertsDir()
	baseDomain := getBaseDomain(domain)
	hosts := []string{baseDomain, "*." + baseDomain}

	ml, err := newLib()
	if err != nil {
		return nil
	}

	cert, err := ml.CertFile(hosts, certsDir)
	if err != nil {
		return nil
	}

	if cert.CertFile != "" {
		_ = os.Remove(cert.CertFile)
	}
	if cert.KeyFile != "" {
		_ = os.Remove(cert.KeyFile)
	}

	return nil
}

// getBaseDomain extracts the base domain for wildcard cert generation.
// For "branch.project.test" returns "project.test".
// For "project.test" returns "project.test".
// For non-.test domains, returns the domain as-is.
func getBaseDomain(domain string) string {
	parts := strings.Split(domain, ".")
	if len(parts) <= 2 {
		return domain
	}

	// For .test TLD, extract the last two meaningful parts
	if parts[len(parts)-1] == "test" && len(parts) > 2 {
		return strings.Join(parts[len(parts)-2:], ".")
	}

	return domain
}
