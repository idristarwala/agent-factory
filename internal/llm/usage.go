package llm

import (
	"sync"
	"time"
)

// UsageRecord is emitted for every call (issue #11 requirement).
type UsageRecord struct {
	Model     string
	Latency   time.Duration
	Input     int
	Output    int
	Total     int
	Status    int
	Error     string
	Timestamp time.Time
}

// Recorder is a simple in-memory recorder for usage.
type Recorder struct {
	mu      sync.Mutex
	Records []UsageRecord
}

func NewRecorder() *Recorder {
	return &Recorder{}
}

func (r *Recorder) Record(u UsageRecord) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u.Timestamp = time.Now()
	r.Records = append(r.Records, u)
}
