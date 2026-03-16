package loop

import (
	"context"

	tool "github.com/benaskins/axon-tool"
)

// Message represents a single message in a conversation.
type Message struct {
	Role      string
	Content   string
	Thinking  string
	ToolCalls []ToolCall
}

// ToolCall represents an LLM's decision to invoke a tool.
type ToolCall struct {
	Name      string
	Arguments map[string]any
}

// Request is a provider-agnostic request to an LLM.
type Request struct {
	Model         string
	Messages      []Message
	Tools         []tool.ToolDef
	Stream        bool
	Think         *bool
	Options       map[string]any
	MaxIterations int // Maximum tool-call loop iterations. Defaults to 20 if 0.
	MaxTokens     int // Maximum estimated token budget for messages. 0 means no limit.
}

// Response is a provider-agnostic streamed response chunk from an LLM.
type Response struct {
	Content   string
	Thinking  string
	Done      bool
	ToolCalls []ToolCall
}

// LLMClient abstracts communication with an LLM backend.
// Implementations translate to/from provider-specific APIs
// (e.g. Ollama, OpenAI, Anthropic).
type LLMClient interface {
	Chat(ctx context.Context, req *Request, fn func(Response) error) error
}

// Event is a streaming event emitted by Stream. Consumers receive these
// on a channel instead of registering callbacks.
type Event struct {
	// Exactly one of these is set per event.
	Token    string            // incremental content token
	Thinking string            // incremental thinking token
	ToolUse  *ToolUseEvent     // a tool was invoked
	Done     *DoneEvent        // the loop completed
	Err      error             // the loop failed
}

// ToolUseEvent is emitted when the LLM invokes a tool.
type ToolUseEvent struct {
	Name string
	Args map[string]any
}

// DoneEvent is emitted when the loop completes successfully.
type DoneEvent struct {
	Content    string
	Thinking   string
	DurationMs int64
}
