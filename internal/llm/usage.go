package llm

import (
	"time"
)

// UsageRecord is emitted for every call (issue #11 requirement).
type UsageRecord struct {
	Model    string
	Latency  time.Duration
	Input    int
	Output   int
	Status   int
	Error    string
	Timestamp time.Time
}

// Recorder is a simple in-memory recorder for usage.
type Recorder struct {
	Records []UsageRecord
}

func NewRecorder() *Recorder {
	return &Recorder{}
}

func (r *Recorder) Record(u UsageRecord) {
	u.Timestamp = time.Now()
	r.Records = append(r.Records, u)
}
