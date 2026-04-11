package mcpdev

import (
	"fmt"
	"path/filepath"
	"strings"
)

// resolveUnderDataDir returns an absolute path inside root for a user-provided relative path.
func resolveUnderDataDir(root, rel string) (string, error) {
	root = filepath.Clean(root)
	if root == "" || root == "." {
		return "", fmt.Errorf("invalid data root")
	}
	rel = strings.TrimSpace(rel)
	if rel == "" || rel == "." {
		return root, nil
	}
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("path must be relative to data directory")
	}
	if strings.Contains(rel, "..") {
		return "", fmt.Errorf("path must not contain parent segments")
	}
	joined := filepath.Join(root, filepath.FromSlash(rel))
	joined = filepath.Clean(joined)
	rrel, err := filepath.Rel(root, joined)
	if err != nil {
		return "", fmt.Errorf("path escapes data directory: %w", err)
	}
	if rrel == ".." || strings.HasPrefix(rrel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes data directory")
	}
	return joined, nil
}
