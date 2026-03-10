package loop_test

import (
	"context"
	"testing"

	loop "github.com/benaskins/axon-loop"
)

func TestMessageFields(t *testing.T) {
	msg := loop.Message{
		Role:    loop.RoleAssistant,
		Content: "Hello!",
		Thinking: "Let me think...",
		ToolCalls: []loop.ToolCall{
			{ID: "call_1", Name: "web_search", Arguments: map[string]any{"query": "go"}},
		},
	}

	if msg.Role != loop.RoleAssistant {
		t.Errorf("Role = %q, want %q", msg.Role, loop.RoleAssistant)
	}
	if len(msg.ToolCalls) != 1 {
		t.Fatalf("got %d tool calls, want 1", len(msg.ToolCalls))
	}
	if msg.ToolCalls[0].Name != "web_search" {
		t.Errorf("ToolCall name = %q, want %q", msg.ToolCalls[0].Name, "web_search")
	}
	if msg.ToolCalls[0].ID != "call_1" {
		t.Errorf("ToolCall ID = %q, want %q", msg.ToolCalls[0].ID, "call_1")
	}
	if msg.ToolCalls[0].Arguments["query"] != "go" {
		t.Errorf("ToolCall query = %v, want %q", msg.ToolCalls[0].Arguments["query"], "go")
	}
}

func TestRequestFields(t *testing.T) {
	req := loop.Request{
		Model: "llama3",
		Messages: []loop.Message{
			{Role: loop.RoleUser, Content: "Hi"},
		},
		Stream: true,
		Options: map[string]any{"temperature": 0.7},
	}

	if req.Model != "llama3" {
		t.Errorf("Model = %q, want %q", req.Model, "llama3")
	}
	if !req.Stream {
		t.Error("Stream should be true")
	}
	if len(req.Messages) != 1 {
		t.Fatalf("got %d messages, want 1", len(req.Messages))
	}
}

func TestResponseFields(t *testing.T) {
	resp := loop.Response{
		Content:  "Here you go",
		Thinking: "Processing...",
		Done:     true,
		ToolCalls: []loop.ToolCall{
			{ID: "call_42", Name: "current_time", Arguments: map[string]any{}},
		},
	}

	if !resp.Done {
		t.Error("Done should be true")
	}
	if resp.Content != "Here you go" {
		t.Errorf("Content = %q, want %q", resp.Content, "Here you go")
	}
	if resp.ToolCalls[0].ID != "call_42" {
		t.Errorf("ToolCall ID = %q, want %q", resp.ToolCalls[0].ID, "call_42")
	}
}

func TestRoleConstants(t *testing.T) {
	tests := []struct {
		role loop.Role
		want string
	}{
		{loop.RoleSystem, "system"},
		{loop.RoleUser, "user"},
		{loop.RoleAssistant, "assistant"},
		{loop.RoleTool, "tool"},
	}
	for _, tt := range tests {
		if string(tt.role) != tt.want {
			t.Errorf("Role = %q, want %q", tt.role, tt.want)
		}
	}
}

// stubClient implements LLMClient for testing.
type stubClient struct {
	responses []loop.Response
}

func (s *stubClient) Chat(ctx context.Context, req *loop.Request, fn func(loop.Response) error) error {
	for _, resp := range s.responses {
		if err := fn(resp); err != nil {
			return err
		}
	}
	return nil
}

func TestLLMClientInterface(t *testing.T) {
	client := &stubClient{
		responses: []loop.Response{
			{Content: "Hello ", Done: false},
			{Content: "World!", Done: true},
		},
	}

	var c loop.LLMClient = client
	var collected string

	err := c.Chat(context.Background(), &loop.Request{
		Model:    "test",
		Messages: []loop.Message{{Role: loop.RoleUser, Content: "Hi"}},
	}, func(resp loop.Response) error {
		collected += resp.Content
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}
	if collected != "Hello World!" {
		t.Errorf("got %q, want %q", collected, "Hello World!")
	}
}
