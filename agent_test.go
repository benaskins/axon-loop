package loop_test

import (
	"context"
	"testing"

	loop "github.com/benaskins/axon-loop"
)

func TestMessageFields(t *testing.T) {
	msg := loop.Message{
		Role:    "assistant",
		Content: "Hello!",
		Thinking: "Let me think...",
		ToolCalls: []loop.ToolCall{
			{Name: "web_search", Arguments: map[string]any{"query": "go"}},
		},
	}

	if msg.Role != "assistant" {
		t.Errorf("Role = %q, want %q", msg.Role, "assistant")
	}
	if len(msg.ToolCalls) != 1 {
		t.Fatalf("got %d tool calls, want 1", len(msg.ToolCalls))
	}
	if msg.ToolCalls[0].Name != "web_search" {
		t.Errorf("ToolCall name = %q, want %q", msg.ToolCalls[0].Name, "web_search")
	}
	if msg.ToolCalls[0].Arguments["query"] != "go" {
		t.Errorf("ToolCall query = %v, want %q", msg.ToolCalls[0].Arguments["query"], "go")
	}
}

func TestChatRequestFields(t *testing.T) {
	req := loop.ChatRequest{
		Model: "llama3",
		Messages: []loop.Message{
			{Role: "user", Content: "Hi"},
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

func TestChatResponseFields(t *testing.T) {
	resp := loop.ChatResponse{
		Content:  "Here you go",
		Thinking: "Processing...",
		Done:     true,
		ToolCalls: []loop.ToolCall{
			{Name: "current_time", Arguments: map[string]any{}},
		},
	}

	if !resp.Done {
		t.Error("Done should be true")
	}
	if resp.Content != "Here you go" {
		t.Errorf("Content = %q, want %q", resp.Content, "Here you go")
	}
}

// stubClient implements LLMClient for testing.
type stubClient struct {
	responses []loop.ChatResponse
}

func (s *stubClient) Chat(ctx context.Context, req *loop.ChatRequest, fn func(loop.ChatResponse) error) error {
	for _, resp := range s.responses {
		if err := fn(resp); err != nil {
			return err
		}
	}
	return nil
}

func TestLLMClientInterface(t *testing.T) {
	client := &stubClient{
		responses: []loop.ChatResponse{
			{Content: "Hello ", Done: false},
			{Content: "World!", Done: true},
		},
	}

	var c loop.LLMClient = client
	var collected string

	err := c.Chat(context.Background(), &loop.ChatRequest{
		Model:    "test",
		Messages: []loop.Message{{Role: "user", Content: "Hi"}},
	}, func(resp loop.ChatResponse) error {
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
