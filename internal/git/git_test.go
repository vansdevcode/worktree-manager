package git

import (
	"testing"
)

func TestConvertGitHubFormat(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "GitHub owner/repo format",
			input:    "owner/repo",
			expected: "git@github.com:owner/repo.git",
		},
		{
			name:     "Already git@ format",
			input:    "git@github.com:owner/repo.git",
			expected: "git@github.com:owner/repo.git",
		},
		{
			name:     "HTTPS URL",
			input:    "https://github.com/owner/repo",
			expected: "https://github.com/owner/repo",
		},
		{
			name:     "HTTPS URL with .git",
			input:    "https://github.com/owner/repo.git",
			expected: "https://github.com/owner/repo.git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertGitHubFormat(tt.input)
			if result != tt.expected {
				t.Errorf("ConvertGitHubFormat(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
