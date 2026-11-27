package secrets

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// DefaultDir is the industry-standard location for mounted secrets
// (Kubernetes, Docker secrets, etc.). Override with SECRETS_DIR.
const DefaultDir = "/etc/secrets"

// Loader reads secrets from a single directory.
// Files are named after the secret (e.g. factory-github-token).
// Values are cached and re-read when the file mtime changes.
type Loader struct {
	Dir string

	mu    sync.RWMutex
	cache map[string]cached
}

type cached struct {
	value string
	mod   int64
}

// New returns a Loader. Uses SECRETS_DIR if set, otherwise /etc/secrets.
func New() *Loader {
	dir := os.Getenv("SECRETS_DIR")
	if dir == "" {
		dir = DefaultDir
	}
	return &Loader{Dir: dir, cache: make(map[string]cached)}
}

// Get returns the contents of <Dir>/<name>, trimmed.
// Falls back to the environment variable of the same name (uppercased, - → _)
// if the file does not exist.
func (l *Loader) Get(name string) (string, error) {
	path := filepath.Join(l.Dir, name)

	info, err := os.Stat(path)
	if err != nil {
		// Fallback to env
		envName := strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
		if v := os.Getenv(envName); v != "" {
			return v, nil
		}
		return "", fmt.Errorf("secret %s not found in %s or env %s", name, l.Dir, envName)
	}

	l.mu.RLock()
	c, ok := l.cache[name]
	l.mu.RUnlock()
	if ok && c.mod == info.ModTime().UnixNano() {
		return c.value, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	val := strings.TrimSpace(string(data))

	l.mu.Lock()
	l.cache[name] = cached{value: val, mod: info.ModTime().UnixNano()}
	l.mu.Unlock()
	return val, nil
}
