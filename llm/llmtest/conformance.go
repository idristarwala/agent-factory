// Package llmtest provides the shared gateway contract tests for adapter authors.
package llmtest

import (
	"testing"

	core "github.com/idristarwala/agent-factory/internal/llm"
	"github.com/idristarwala/agent-factory/llm"
)

type Harness interface{ Client() llm.Client }

func RunConformance(t *testing.T, h Harness) {
	t.Helper()
	core.RunConformance(t, h)
	if p, ok := h.(*ProtocolHarness); ok {
		RunHTTPConformance(t, p.NewClient)
		RunStructuredConformance(t, p.Protocol, p.NewClient)
		t.Run("conversation_and_stream", func(t *testing.T) { checkConversation(t, p) })
	}
}
