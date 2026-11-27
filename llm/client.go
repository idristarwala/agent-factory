// Package llm exposes the gateway client contract for external adapters.
package llm

import core "github.com/idristarwala/agent-factory/internal/llm"

type (
	Client         = core.Client
	ChatRequest    = core.ChatRequest
	ChatResponse   = core.ChatResponse
	Message        = core.Message
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
