package loop_test

import (
	"context"
	"fmt"

	loop "github.com/benaskins/axon-loop"
	tool "github.com/benaskins/axon-tool"
)

// A sliding window keeps the system prompt plus the last n messages,
// discarding older history as the conversation grows.
func ExampleSlidingWindow() {
	strategy := loop.SlidingWindow(10)

	messages := []loop.Message{
		{Role: loop.RoleSystem, Content: "You are a helpful assistant."},
		{Role: loop.RoleUser, Content: "Hello"},
		{Role: loop.RoleAssistant, Content: "Hi there!"},
		{Role: loop.RoleUser, Content: "What is Go?"},
	}

	trimmed := strategy.Trim(messages)
	fmt.Println(trimmed[0].Role, "message preserved")
	fmt.Println(len(trimmed)-1, "conversation messages kept")
	// Output:
	// system message preserved
	// 3 conversation messages kept
}

// A token budget trims older messages so the total estimated token count
// stays within the budget. The system prompt is always preserved.
func ExampleTokenBudget() {
	strategy := loop.TokenBudget(4096)

	messages := []loop.Message{
		{Role: loop.RoleSystem, Content: "You are a helpful assistant."},
		{Role: loop.RoleUser, Content: "Summarise this long document..."},
	}

	trimmed := strategy.Trim(messages)
	fmt.Println(len(trimmed), "messages after trim")
	// Output:
	// 2 messages after trim
}

// RunConfig assembles everything needed for a conversation loop: an LLM
// client, the initial request, tool definitions, and a context strategy.
// This is the primary struct consumers construct.
func ExampleRunConfig() {
	// In production, use an axon-talk adapter (e.g. talk.NewOllamaClient).
	var client loop.LLMClient

	// Define tools the model can call.
	tools := map[string]tool.ToolDef{
		"get_weather": {
			Name:        "get_weather",
			Description: "Get current weather for a city",
			Parameters: tool.ParameterSchema{
				Type:     "object",
				Required: []string{"city"},
				Properties: map[string]tool.PropertySchema{
					"city": {Type: "string", Description: "City name"},
				},
			},
			Execute: func(ctx *tool.ToolContext, args map[string]any) tool.ToolResult {
				city, _ := args["city"].(string)
				return tool.ToolResult{Content: fmt.Sprintf("22°C and sunny in %s", city)}
			},
		},
	}

	cfg := loop.RunConfig{
		Client: client,
		Request: &loop.Request{
			Model:  "claude-sonnet-4-20250514",
			Stream: true,
			Messages: []loop.Message{
				{Role: loop.RoleSystem, Content: "You are a weather assistant."},
				{Role: loop.RoleUser, Content: "What's the weather in Melbourne?"},
			},
			MaxIterations: 5,
		},
		Tools:   tools,
		Context: loop.SlidingWindow(20),
		Callbacks: loop.Callbacks{
			OnToken: func(token string) {
				fmt.Print(token)
			},
		},
	}

	_, _ = loop.Run(context.Background(), cfg)
}
