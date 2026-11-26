package obs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

type traceKey struct{}

func WithTrace(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, traceKey{}, id)
}

func TraceFrom(ctx context.Context) string {
	if v, ok := ctx.Value(traceKey{}).(string); ok {
		return v
	}
	return ""
}

type Span struct {
	TraceID  string         `json:"trace_id"`
	SpanID   string         `json:"span_id"`
	ParentID string         `json:"parent_id,omitempty"`
	Name     string         `json:"name"`
	Start    time.Time      `json:"start"`
	End      time.Time      `json:"end,omitempty"`
	Status   string         `json:"status"`
	Attrs    map[string]any `json:"attrs,omitempty"`
	Error    string         `json:"error,omitempty"`
}

type Logger struct {
	out io.Writer
	mu  sync.Mutex
}

func New() *Logger { return &Logger{out: os.Stdout} }

func (l *Logger) Info(ctx context.Context, msg string, attrs map[string]any) {
	l.emit("info", ctx, msg, attrs, nil)
}

func (l *Logger) Error(ctx context.Context, msg string, err error, attrs map[string]any) {
	l.emit("error", ctx, msg, attrs, err)
}

func (l *Logger) Span(ctx context.Context, name string) *spanHandle {
	return &spanHandle{
		l: l,
		s: Span{
			TraceID: TraceFrom(ctx),
			SpanID:  newID(),
			Name:    name,
			Start:   time.Now(),
			Status:  "ok",
			Attrs:   map[string]any{},
		},
	}
}

type spanHandle struct {
	l *Logger
	s Span
}

func (h *spanHandle) Attr(k string, v any) *spanHandle {
	h.s.Attrs[k] = v
	return h
}

func (h *spanHandle) End(err error) {
	h.s.End = time.Now()
	if err != nil {
		h.s.Status = "error"
		h.s.Error = err.Error()
	}
	h.l.write(map[string]any{
		"ts":     h.s.End.Format(time.RFC3339Nano),
		"level":  "info",
		"event":  "span",
		"trace":  h.s.TraceID,
		"span":   h.s.SpanID,
		"name":   h.s.Name,
		"dur_ms": h.s.End.Sub(h.s.Start).Milliseconds(),
		"status": h.s.Status,
		"attrs":  h.s.Attrs,
		"error":  h.s.Error,
	})
}

func (l *Logger) emit(level string, ctx context.Context, msg string, attrs map[string]any, err error) {
	e := map[string]any{
		"ts":    time.Now().Format(time.RFC3339Nano),
		"level": level,
		"msg":   msg,
		"trace": TraceFrom(ctx),
	}
	if attrs != nil {
		e["attrs"] = attrs
	}
	if err != nil {
		e["error"] = err.Error()
	}
	l.write(e)
}

func (l *Logger) write(e map[string]any) {
	b, _ := json.Marshal(e)
	l.mu.Lock()
	fmt.Fprintln(l.out, string(b))
	l.mu.Unlock()
}

func newID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
