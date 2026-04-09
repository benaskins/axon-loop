package loop

import (
	talk "github.com/benaskins/axon-talk"
)

// Type aliases re-exported from axon-talk for backwards compatibility
// within this package. Internal code uses these directly; external
// consumers should migrate to axon-talk types.
type (
	Role     = talk.Role
	Message  = talk.Message
	ToolCall = talk.ToolCall
	Request  = talk.Request
	Response = talk.Response
	Usage    = talk.Usage

	LLMClient = talk.LLMClient
)

const (
	RoleSystem    = talk.RoleSystem
	RoleUser      = talk.RoleUser
	RoleAssistant = talk.RoleAssistant
	RoleTool      = talk.RoleTool
)

// Event is a streaming event emitted by Stream. Consumers receive these
// on a channel instead of registering callbacks.
type Event struct {
	// Exactly one of these is set per event.
	Token    string        // incremental content token
	Thinking string        // incremental thinking token
	ToolUse  *ToolUseEvent // a tool was invoked
	Trim     *TrimEvent    // context was trimmed
	Done     *DoneEvent    // the loop completed
	Err      error         // the loop failed
}

// TrimEvent is emitted when the context strategy drops messages.
type TrimEvent struct {
	Dropped []Message
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
