// Package llm exposes the gateway client contract for external adapters.
package llm

import (
	"context"
	core "github.com/idristarwala/agent-factory/internal/llm"
	"time"
)

type (
	Client         = core.Client
	ChatRequest    = core.ChatRequest
	ChatResponse   = core.ChatResponse
	Message        = core.Message
	ImagePart      = core.ImagePart
	Tier           = core.Tier
	Role           = core.Role
	ToolSpec       = core.ToolSpec
	ToolCall       = core.ToolCall
	Stream         = core.Stream
	StreamEvent    = core.StreamEvent
	Usage          = core.Usage
	Error          = core.Error
	AdapterFactory = core.AdapterFactory
	OpenAIAdapter  = core.OpenAIAdapter
	OpenAIEnvelope = core.OpenAIEnvelope
)

const (
	TierCheap    = core.TierCheap
	TierFrontier = core.TierFrontier
	TierCouncil  = core.TierCouncil
	RoleWorker   = core.RoleWorker
	RoleSteward  = core.RoleSteward
	RoleJudge    = core.RoleJudge
)

var ErrStreamingUnsupported = core.ErrStreamingUnsupported

func NewOpenAIAdapter(baseURL, key string, headers map[string]string) *OpenAIAdapter {
	return core.NewOpenAIAdapter(baseURL, key, headers)
}

func Register(name string, factory AdapterFactory) { core.Register(name, factory) }

func New(name string, config map[string]string) (Client, error) { return core.New(name, config) }

// NewTokenRefresher wraps a gateway with cached credential rotation.
// The supplied client can implement SetAPIKey(string) to install a new key.
func NewTokenRefresher(client Client, refresh func(context.Context) (string, error), interval time.Duration) Client {
	return core.NewTokenRefresher(client, refresh, interval)
}

// CollectStream assembles text, function calls, final usage and finish reason.
func CollectStream(stream Stream) (*ChatResponse, error) { return core.CollectStream(stream) }
