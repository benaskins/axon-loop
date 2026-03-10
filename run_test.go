package loop_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	loop "github.com/benaskins/axon-loop"
	tool "github.com/benaskins/axon-tool"
)

func TestRunSimpleChat(t *testing.T) {
	client := &stubClient{
		responses: []loop.Response{
			{Content: "Hello there!", Done: true},
		},
	}

	var tokens []string
	var doneCalled bool

	result, err := loop.Run(context.Background(), loop.RunConfig{
		Client: client,
		Request: &loop.Request{
			Model:    "test-model",
			Messages: []loop.Message{{Role: loop.RoleUser, Content: "Hi"}},
		},
		Callbacks: loop.Callbacks{
			OnToken: func(token string) {
				tokens = append(tokens, token)
			},
			OnDone: func(durationMs int64) {
				doneCalled = true
			},
		},
	})

	if err != nil {
		t.Fatal(err)
	}
	if result.Content != "Hello there!" {
		t.Errorf("Content = %q, want %q", result.Content, "Hello there!")
	}
	if !doneCalled {
		t.Error("OnDone was not called")
	}
	if len(tokens) == 0 {
		t.Error("expected at least one OnToken call")
	}
}

func TestRunWithToolCall(t *testing.T) {
	callCount := 0
	client := &multiTurnClient{
		turns: [][]loop.Response{
			// First turn: model calls a tool
			{
				{
					Content: "",
					ToolCalls: []loop.ToolCall{
						{ID: "call_1", Name: "current_time", Arguments: map[string]any{}},
					},
					Done: true,
				},
			},
			// Second turn: model responds with final answer
			{
				{Content: "It is 3pm.", Done: true},
			},
		},
	}

	tools := map[string]tool.ToolDef{
		"current_time": {
			Name: "current_time",
			Execute: func(ctx *tool.ToolContext, args map[string]any) tool.ToolResult {
				callCount++
				return tool.ToolResult{Content: "Current time: Monday, 3:00 PM"}
			},
		},
	}

	var toolUses []string
	result, err := loop.Run(context.Background(), loop.RunConfig{
		Client: client,
		Request: &loop.Request{
			Model:    "test-model",
			Messages: []loop.Message{{Role: loop.RoleUser, Content: "What time is it?"}},
		},
		Tools:   tools,
		ToolCtx: &tool.ToolContext{Ctx: context.Background()},
		Callbacks: loop.Callbacks{
			OnToolUse: func(name string, args map[string]any) {
				toolUses = append(toolUses, name)
			},
		},
	})

	if err != nil {
		t.Fatal(err)
	}
	if callCount != 1 {
		t.Errorf("tool executed %d times, want 1", callCount)
	}
	if len(toolUses) != 1 || toolUses[0] != "current_time" {
		t.Errorf("OnToolUse calls = %v, want [current_time]", toolUses)
	}
	if result.Content != "It is 3pm." {
		t.Errorf("Content = %q, want %q", result.Content, "It is 3pm.")
	}
}

func TestRunNoTools(t *testing.T) {
	client := &stubClient{
		responses: []loop.Response{
			{Content: "Just chatting.", Done: true},
		},
	}

	result, err := loop.Run(context.Background(), loop.RunConfig{
		Client: client,
		Request: &loop.Request{
			Model:    "test-model",
			Messages: []loop.Message{{Role: loop.RoleUser, Content: "Hello"}},
		},
	})

	if err != nil {
		t.Fatal(err)
	}
	if result.Content != "Just chatting." {
		t.Errorf("Content = %q, want %q", result.Content, "Just chatting.")
	}
}

func TestRunWithThinking(t *testing.T) {
	client := &stubClient{
		responses: []loop.Response{
			{Thinking: "Let me consider...", Done: false},
			{Content: "Here's my answer.", Done: true},
		},
	}

	var thinkingTokens []string
	result, err := loop.Run(context.Background(), loop.RunConfig{
		Client: client,
		Request: &loop.Request{
			Model:    "test-model",
			Messages: []loop.Message{{Role: loop.RoleUser, Content: "Think about this"}},
		},
		Callbacks: loop.Callbacks{
			OnThinking: func(token string) {
				thinkingTokens = append(thinkingTokens, token)
			},
		},
	})

	if err != nil {
		t.Fatal(err)
	}
	if result.Thinking != "Let me consider..." {
		t.Errorf("Thinking = %q, want %q", result.Thinking, "Let me consider...")
	}
	if len(thinkingTokens) == 0 {
		t.Error("expected OnThinking callback")
	}
}

func TestRunPassesToolsToClient(t *testing.T) {
	var receivedTools []tool.ToolDef
	client := &spyClient{
		onChat: func(req *loop.Request) {
			receivedTools = req.Tools
		},
		responses: []loop.Response{
			{Content: "ok", Done: true},
		},
	}

	tools := map[string]tool.ToolDef{
		"current_time": tool.CurrentTimeTool(),
	}

	_, err := loop.Run(context.Background(), loop.RunConfig{
		Client: client,
		Request: &loop.Request{
			Model:    "test",
			Messages: []loop.Message{{Role: loop.RoleUser, Content: "time?"}},
		},
		Tools:   tools,
		ToolCtx: &tool.ToolContext{Ctx: context.Background()},
	})

	if err != nil {
		t.Fatal(err)
	}
	if len(receivedTools) != 1 {
		t.Fatalf("expected 1 tool in request, got %d", len(receivedTools))
	}
	if receivedTools[0].Name != "current_time" {
		t.Errorf("tool name = %q, want %q", receivedTools[0].Name, "current_time")
	}
}

// spyClient records the ChatRequest for inspection.
type spyClient struct {
	onChat    func(req *loop.Request)
	responses []loop.Response
}

func (s *spyClient) Chat(ctx context.Context, req *loop.Request, fn func(loop.Response) error) error {
	if s.onChat != nil {
		s.onChat(req)
	}
	for _, resp := range s.responses {
		if err := fn(resp); err != nil {
			return err
		}
	}
	return nil
}

func TestRunMaxIterationsExceeded(t *testing.T) {
	// Client that always returns a tool call, forcing infinite loop
	client := &alwaysToolCallClient{}

	tools := map[string]tool.ToolDef{
		"noop": {
			Name: "noop",
			Execute: func(ctx *tool.ToolContext, args map[string]any) tool.ToolResult {
				return tool.ToolResult{Content: "ok"}
			},
		},
	}

	_, err := loop.Run(context.Background(), loop.RunConfig{
		Client: client,
		Request: &loop.Request{
			Model:         "test",
			Messages:      []loop.Message{{Role: loop.RoleUser, Content: "loop"}},
			MaxIterations: 3,
		},
		Tools:   tools,
		ToolCtx: &tool.ToolContext{Ctx: context.Background()},
	})

	if err == nil {
		t.Fatal("expected error for max iterations exceeded, got nil")
	}
	if !strings.Contains(err.Error(), "max iterations") {
		t.Errorf("error = %q, want it to contain 'max iterations'", err.Error())
	}
}

func TestRunUnknownToolCall(t *testing.T) {
	client := &multiTurnClient{
		turns: [][]loop.Response{
			// First turn: model calls a tool that doesn't exist
			{
				{
					ToolCalls: []loop.ToolCall{
						{ID: "call_1", Name: "nonexistent_tool", Arguments: map[string]any{}},
					},
					Done: true,
				},
			},
			// Second turn: model responds with final answer
			{
				{Content: "Sorry about that.", Done: true},
			},
		},
	}

	tools := map[string]tool.ToolDef{
		"real_tool": {
			Name: "real_tool",
			Execute: func(ctx *tool.ToolContext, args map[string]any) tool.ToolResult {
				return tool.ToolResult{Content: "ok"}
			},
		},
	}

	result, err := loop.Run(context.Background(), loop.RunConfig{
		Client: client,
		Request: &loop.Request{
			Model:    "test",
			Messages: []loop.Message{{Role: loop.RoleUser, Content: "call something"}},
		},
		Tools:   tools,
		ToolCtx: &tool.ToolContext{Ctx: context.Background()},
	})

	if err != nil {
		t.Fatal(err)
	}
	if result.Content != "Sorry about that." {
		t.Errorf("Content = %q, want %q", result.Content, "Sorry about that.")
	}
}

func TestRunChatClientError(t *testing.T) {
	client := &errorClient{err: fmt.Errorf("connection refused")}

	_, err := loop.Run(context.Background(), loop.RunConfig{
		Client: client,
		Request: &loop.Request{
			Model:    "test",
			Messages: []loop.Message{{Role: loop.RoleUser, Content: "Hi"}},
		},
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "connection refused") {
		t.Errorf("error = %q, want it to contain 'connection refused'", err.Error())
	}
}

func TestRunPanickingToolRecovery(t *testing.T) {
	client := &multiTurnClient{
		turns: [][]loop.Response{
			// First turn: model calls a tool that panics
			{
				{
					ToolCalls: []loop.ToolCall{
						{ID: "call_1", Name: "bad_tool", Arguments: map[string]any{}},
					},
					Done: true,
				},
			},
			// Second turn: model responds after seeing panic error
			{
				{Content: "The tool failed.", Done: true},
			},
		},
	}

	tools := map[string]tool.ToolDef{
		"bad_tool": {
			Name: "bad_tool",
			Execute: func(ctx *tool.ToolContext, args map[string]any) tool.ToolResult {
				panic("segfault simulation")
			},
		},
	}

	result, err := loop.Run(context.Background(), loop.RunConfig{
		Client: client,
		Request: &loop.Request{
			Model:    "test",
			Messages: []loop.Message{{Role: loop.RoleUser, Content: "use the bad tool"}},
		},
		Tools:   tools,
		ToolCtx: &tool.ToolContext{Ctx: context.Background()},
	})

	if err != nil {
		t.Fatalf("Run should not fail when a tool panics, got: %v", err)
	}
	if result.Content != "The tool failed." {
		t.Errorf("Content = %q, want %q", result.Content, "The tool failed.")
	}
}

func TestRunToolCallIDPropagation(t *testing.T) {
	var capturedMessages []loop.Message
	client := &multiTurnClient{
		turns: [][]loop.Response{
			{
				{
					ToolCalls: []loop.ToolCall{
						{ID: "call_abc123", Name: "echo", Arguments: map[string]any{"text": "hi"}},
					},
					Done: true,
				},
			},
			{
				{Content: "Done.", Done: true},
			},
		},
	}

	// Use a spy to capture the messages sent on the second turn
	origChat := client.Chat
	client2 := &spyClient{
		onChat: func(req *loop.Request) {
			capturedMessages = req.Messages
		},
	}
	// Wrap: first turn from multiTurnClient, spy captures second turn
	wrapper := &delegatingClient{
		clients: []loop.LLMClient{client, client2},
	}
	_ = origChat // suppress unused

	tools := map[string]tool.ToolDef{
		"echo": {
			Name: "echo",
			Execute: func(ctx *tool.ToolContext, args map[string]any) tool.ToolResult {
				return tool.ToolResult{Content: "echoed"}
			},
		},
	}

	_, _ = loop.Run(context.Background(), loop.RunConfig{
		Client: wrapper,
		Request: &loop.Request{
			Model:    "test",
			Messages: []loop.Message{{Role: loop.RoleUser, Content: "echo something"}},
		},
		Tools:   tools,
		ToolCtx: &tool.ToolContext{Ctx: context.Background()},
	})

	// Find the tool result message in captured messages
	var found bool
	for _, msg := range capturedMessages {
		if msg.Role == loop.RoleTool {
			found = true
			if msg.ToolCallID != "call_abc123" {
				t.Errorf("ToolCallID = %q, want %q", msg.ToolCallID, "call_abc123")
			}
		}
	}
	if !found {
		t.Error("no tool result message found in captured messages")
	}
}

func TestRunContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	client := &cancellingClient{
		cancel: cancel,
	}

	_, err := loop.Run(ctx, loop.RunConfig{
		Client: client,
		Request: &loop.Request{
			Model:    "test",
			Messages: []loop.Message{{Role: loop.RoleUser, Content: "Hi"}},
		},
	})

	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
	if !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("error = %q, want it to contain 'context canceled'", err.Error())
	}
}

func TestRunMultiToolCallsInSingleTurn(t *testing.T) {
	var toolOrder []string
	client := &multiTurnClient{
		turns: [][]loop.Response{
			// Model calls two tools in one turn
			{
				{
					ToolCalls: []loop.ToolCall{
						{ID: "call_1", Name: "tool_a", Arguments: map[string]any{}},
						{ID: "call_2", Name: "tool_b", Arguments: map[string]any{}},
					},
					Done: true,
				},
			},
			// Final answer
			{
				{Content: "Both done.", Done: true},
			},
		},
	}

	tools := map[string]tool.ToolDef{
		"tool_a": {
			Name: "tool_a",
			Execute: func(ctx *tool.ToolContext, args map[string]any) tool.ToolResult {
				toolOrder = append(toolOrder, "a")
				return tool.ToolResult{Content: "result_a"}
			},
		},
		"tool_b": {
			Name: "tool_b",
			Execute: func(ctx *tool.ToolContext, args map[string]any) tool.ToolResult {
				toolOrder = append(toolOrder, "b")
				return tool.ToolResult{Content: "result_b"}
			},
		},
	}

	result, err := loop.Run(context.Background(), loop.RunConfig{
		Client: client,
		Request: &loop.Request{
			Model:    "test",
			Messages: []loop.Message{{Role: loop.RoleUser, Content: "use both tools"}},
		},
		Tools:   tools,
		ToolCtx: &tool.ToolContext{Ctx: context.Background()},
	})

	if err != nil {
		t.Fatal(err)
	}
	if result.Content != "Both done." {
		t.Errorf("Content = %q, want %q", result.Content, "Both done.")
	}
	if len(toolOrder) != 2 {
		t.Fatalf("expected 2 tool calls, got %d", len(toolOrder))
	}
	if toolOrder[0] != "a" || toolOrder[1] != "b" {
		t.Errorf("tool execution order = %v, want [a b]", toolOrder)
	}
}

// alwaysToolCallClient always returns a tool call on every Chat invocation.
type alwaysToolCallClient struct{}

func (a *alwaysToolCallClient) Chat(ctx context.Context, req *loop.Request, fn func(loop.Response) error) error {
	return fn(loop.Response{
		ToolCalls: []loop.ToolCall{
			{Name: "noop", Arguments: map[string]any{}},
		},
		Done: true,
	})
}

// errorClient always returns an error.
type errorClient struct {
	err error
}

func (e *errorClient) Chat(ctx context.Context, req *loop.Request, fn func(loop.Response) error) error {
	return e.err
}

// multiTurnClient simulates a client that returns different responses on each call.
type multiTurnClient struct {
	turns [][]loop.Response
	call  int
}

func (m *multiTurnClient) Chat(ctx context.Context, req *loop.Request, fn func(loop.Response) error) error {
	if m.call >= len(m.turns) {
		return nil
	}
	responses := m.turns[m.call]
	m.call++
	for _, resp := range responses {
		if err := fn(resp); err != nil {
			return err
		}
	}
	return nil
}

// cancellingClient cancels the context during Chat, simulating mid-loop cancellation.
type cancellingClient struct {
	cancel context.CancelFunc
}

func (c *cancellingClient) Chat(ctx context.Context, req *loop.Request, fn func(loop.Response) error) error {
	c.cancel()
	return ctx.Err()
}

// delegatingClient delegates to a sequence of clients, one per call.
type delegatingClient struct {
	clients []loop.LLMClient
	call    int
}

func (d *delegatingClient) Chat(ctx context.Context, req *loop.Request, fn func(loop.Response) error) error {
	if d.call >= len(d.clients) {
		return nil
	}
	c := d.clients[d.call]
	d.call++
	return c.Chat(ctx, req, fn)
}
