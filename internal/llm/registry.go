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
