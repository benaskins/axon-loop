//go:build ignore

// Example shows a minimal conversation loop setup.
//
// This is a sketch — you need an LLMClient implementation (e.g. from
// axon-talk) to actually run it. The wiring pattern is the point.
package main

import (
	"context"
	"fmt"
	"log"

	loop "github.com/benaskins/axon-loop"
	tool "github.com/benaskins/axon-tool"
)

func main() {
	ctx := context.Background()

	// 1. Create your LLM client (axon-talk provides Ollama, or roll your own).
	var client loop.LLMClient // = talk.NewOllamaClient("http://localhost:11434")

	// 2. Define tools the agent can call.
	tools := map[string]tool.ToolDef{
		"greet": {
			Name:        "greet",
			Description: "Greet someone by name",
			Parameters: tool.ParameterSchema{
				Type: "object",
				Properties: map[string]tool.PropertySchema{
					"name": {Type: "string", Description: "Name to greet"},
				},
				Required: []string{"name"},
			},
			Execute: func(tc *tool.ToolContext, args map[string]any) tool.ToolResult {
				name, _ := args["name"].(string)
				return tool.ToolResult{Content: fmt.Sprintf("Hello, %s!", name)}
			},
		},
	}

	// 3. Build the request.
	req := &loop.Request{
		Model: "llama3",
		Messages: []loop.Message{
			{Role: "system", Content: "You are a helpful assistant. Use tools when appropriate."},
			{Role: "user", Content: "Please greet Alice."},
		},
		Stream: true,
	}

	// 4. Run the loop with streaming callbacks.
	result, err := loop.Run(ctx, client, req, tools, nil, loop.Callbacks{
		OnToken:   func(t string) { fmt.Print(t) },
		OnToolUse: func(name string, args map[string]any) { fmt.Printf("\n[tool] %s %v\n", name, args) },
		OnDone:    func(ms int64) { fmt.Printf("\n[done in %dms]\n", ms) },
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("\nFinal:", result.Content)
}
