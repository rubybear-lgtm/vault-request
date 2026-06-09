package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Store is the interface for persisting secrets.
type Store interface {
	// Save writes a key-value pair to the backend.
	Save(key, value string) error
}

// EnvStore writes secrets to a .env file in key=value format.
type EnvStore struct {
	Path string // absolute path to the .env file
}

// NewEnvStore creates an EnvStore backed by the given file path.
// Parent directories are created if they don't exist.
func NewEnvStore(path string) (*EnvStore, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}
	dir := filepath.Dir(abs)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("create dir %s: %w", dir, err)
	}
	return &EnvStore{Path: abs}, nil
}

// Save writes key=value to the .env file. If the key already exists, it is
// overwritten. The value is shell-quoted to survive basic shell parsing.
func (e *EnvStore) Save(key, value string) error {
	line := fmt.Sprintf(`%s="%s"`, key, shellEscape(value))

	existing, err := os.ReadFile(e.Path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read %s: %w", e.Path, err)
	}

	// Check if key already exists — if so, replace in place.
	content := string(existing)
	prefix := key + "="
	replaced := false
	var lines []string
	for _, l := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(l)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, prefix) {
			lines = append(lines, line)
			replaced = true
		} else {
			lines = append(lines, l)
		}
	}
	if !replaced {
		lines = append(lines, line)
	}

	// Rebuild with trailing newline.
	output := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(e.Path, []byte(output), 0600); err != nil {
		return fmt.Errorf("write %s: %w", e.Path, err)
	}
	return nil
}

// shellEscape escapes a string for safe use inside double-quoted shell values.
// Backslashes, double quotes, dollar signs, and backticks are escaped.
func shellEscape(s string) string {
	replacer := strings.NewReplacer(
		`\`, `\\`,
		`"`, `\"`,
		`$`, `\$`,
		"`", "\\`",
	)
	return replacer.Replace(s)
}
