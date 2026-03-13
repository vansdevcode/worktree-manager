package process

import (
	"os"
	"strings"
	"testing"
)

func TestBuildEnv_Nil(t *testing.T) {
	got := buildEnv(nil)
	if got != nil {
		t.Errorf("buildEnv(nil) = %v, want nil", got)
	}
}

func TestBuildEnv_ExpandsVars(t *testing.T) {
	t.Setenv("DEVTREE_TEST_VAR", "expanded_value")

	extra := map[string]string{
		"LITERAL":  "hello",
		"EXPANDED": "$DEVTREE_TEST_VAR",
		"BRACES":   "${DEVTREE_TEST_VAR}/sub",
		"MISSING":  "$DEVTREE_NONEXISTENT_VAR",
	}

	env := buildEnv(extra)

	lookup := make(map[string]string)
	for _, entry := range env {
		k, v, _ := strings.Cut(entry, "=")
		lookup[k] = v
	}

	if lookup["LITERAL"] != "hello" {
		t.Errorf("LITERAL = %q, want %q", lookup["LITERAL"], "hello")
	}
	if lookup["EXPANDED"] != "expanded_value" {
		t.Errorf("EXPANDED = %q, want %q", lookup["EXPANDED"], "expanded_value")
	}
	if lookup["BRACES"] != "expanded_value/sub" {
		t.Errorf("BRACES = %q, want %q", lookup["BRACES"], "expanded_value/sub")
	}
	if lookup["MISSING"] != "" {
		t.Errorf("MISSING = %q, want empty string", lookup["MISSING"])
	}

	// Should also contain inherited env.
	if lookup["HOME"] == "" && os.Getenv("HOME") != "" {
		t.Error("expected inherited HOME env var")
	}
}
