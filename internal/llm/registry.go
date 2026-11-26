package llm

import (
	"fmt"
	"sync"
)

// AdapterFactory creates a Client from config.
type AdapterFactory func(cfg map[string]string) (Client, error)

var (
	regMu    sync.RWMutex
	registry = map[string]AdapterFactory{}
)

// Register adds an adapter factory. Call from init() in adapter packages.
func Register(name string, f AdapterFactory) {
	regMu.Lock()
	defer regMu.Unlock()
	registry[name] = f
}

// New creates a Client by adapter name.
func New(name string, cfg map[string]string) (Client, error) {
	regMu.RLock()
	f, ok := registry[name]
	regMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown adapter: %s", name)
	}
	return f(cfg)
}

// Adapters returns registered adapter names.
func Adapters() []string {
	regMu.RLock()
	defer regMu.RUnlock()
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	return names
}
