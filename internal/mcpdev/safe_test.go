package mcpdev

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveUnderDataDir(t *testing.T) {
	t.Parallel()
	root := filepath.Join(t.TempDir(), "data")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		rel string
		ok  bool
	}{
		{"", true},
		{".", true},
		{"wal.log", true},
		{"auth/sessions.json", true},
		{"../outside", false},
		{"/etc/passwd", false},
	}
	for _, tc := range cases {
		_, err := resolveUnderDataDir(root, tc.rel)
		if tc.ok && err != nil {
			t.Errorf("rel %q: %v", tc.rel, err)
		}
		if !tc.ok && err == nil {
			t.Errorf("rel %q: expected error", tc.rel)
		}
	}
}
