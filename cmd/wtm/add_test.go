package main

import "testing"

func TestNormalizeRemoteBranch(t *testing.T) {
	tests := []struct {
		name           string
		branchRef      string
		wantLocalName  string
		wantStartPoint string
	}{
		{
			name:           "origin prefix",
			branchRef:      "origin/develop",
			wantLocalName:  "develop",
			wantStartPoint: "origin/develop",
		},
		{
			name:           "origin with nested path",
			branchRef:      "origin/feature/user-auth",
			wantLocalName:  "feature/user-auth",
			wantStartPoint: "origin/feature/user-auth",
		},
		{
			name:           "local branch main",
			branchRef:      "main",
			wantLocalName:  "main",
			wantStartPoint: "",
		},
		{
			name:           "local branch with slash",
			branchRef:      "feature/test",
			wantLocalName:  "feature/test",
			wantStartPoint: "",
		},
		{
			name:           "tag reference",
			branchRef:      "v1.0.0",
			wantLocalName:  "v1.0.0",
			wantStartPoint: "",
		},
		{
			name:           "commit hash",
			branchRef:      "a1b2c3d",
			wantLocalName:  "a1b2c3d",
			wantStartPoint: "",
		},
		{
			name:           "upstream remote",
			branchRef:      "upstream/main",
			wantLocalName:  "upstream/main",
			wantStartPoint: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLocalName, gotStartPoint := normalizeRemoteBranch(tt.branchRef)
			if gotLocalName != tt.wantLocalName {
				t.Errorf("normalizeRemoteBranch() localName = %q, want %q", gotLocalName, tt.wantLocalName)
			}
			if gotStartPoint != tt.wantStartPoint {
				t.Errorf("normalizeRemoteBranch() startPoint = %q, want %q", gotStartPoint, tt.wantStartPoint)
			}
		})
	}
}
