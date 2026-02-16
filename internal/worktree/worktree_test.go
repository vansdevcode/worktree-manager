package worktree

import (
	"testing"
)

func TestGenerateWorktreeDirectory(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple branch name",
			input:    "feature-branch",
			expected: "feature-branch",
		},
		{
			name:     "Branch with slashes",
			input:    "feature/user-auth",
			expected: "feature-user-auth",
		},
		{
			name:     "Mixed case - preserves case",
			input:    "Feature/User-Auth",
			expected: "Feature-User-Auth",
		},
		{
			name:     "With underscores",
			input:    "fix_bug_123",
			expected: "fix-bug-123",
		},
		{
			name:     "With special characters",
			input:    "feature/user@auth#123",
			expected: "feature-userauth123",
		},
		{
			name:     "Multiple consecutive special chars",
			input:    "feature///user---auth",
			expected: "feature-user-auth",
		},
		{
			name:     "Uppercase branch",
			input:    "JUI-751",
			expected: "JUI-751",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateWorktreeDirectory(tt.input)
			if result != tt.expected {
				t.Errorf("GenerateWorktreeDirectory(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
