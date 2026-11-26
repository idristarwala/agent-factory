package obs

import (
	"sync"
	"time"
)

type Metrics struct {
	mu sync.Mutex
	Decisions  int64
	Approved   int64
	Rejected   int64
	LLMCalls   int64
	LLMErrors  int64
	LLMLatency time.Duration
}

func NewMetrics() *Metrics { return &Metrics{} }

func (m *Metrics) Decision(approved bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Decisions++
	if approved {
		m.Approved++
	} else {
		m.Rejected++
	}
}

func (m *Metrics) LLM(latency time.Duration, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.LLMCalls++
	m.LLMLatency += latency
	if err != nil {
		m.LLMErrors++
	}
}

func (m *Metrics) Snapshot() map[string]any {
	m.mu.Lock()
	defer m.mu.Unlock()
	avg := time.Duration(0)
	if m.LLMCalls > 0 {
		avg = m.LLMLatency / time.Duration(m.LLMCalls)
	}
	rejectRate := 0.0
	if m.Decisions > 0 {
		rejectRate = float64(m.Rejected) / float64(m.Decisions)
	}
	return map[string]any{
		"decisions":   m.Decisions,
		"approved":    m.Approved,
		"rejected":    m.Rejected,
		"reject_rate": rejectRate,
		"llm_calls":   m.LLMCalls,
		"llm_errors":  m.LLMErrors,
		"llm_avg_ms":  avg.Milliseconds(),
	}
}
