# axon-loop

> Primitives · Part of the [lamina](https://github.com/benaskins/lamina-mono) workspace

Provider-agnostic conversation loop for LLM-powered agents. Handles message exchange, tool call dispatch, and streaming with no HTTP, persistence, or UI concerns. Bring your own `LLMClient` implementation (e.g. Ollama, OpenAI, Anthropic) and axon-loop drives the send-stream-tool-repeat cycle.

## Getting started

```bash
go get github.com/benaskins/axon-loop@latest
```

Requires Go 1.26+.

```go
result, err := loop.Run(ctx, loop.RunConfig{
    Client: client,
    Request: &loop.Request{
        Model:    "llama3",
        Messages: []loop.Message{{Role: "user", Content: "Hello"}},
        Stream:   true,
    },
    Tools: tools,
    Callbacks: loop.Callbacks{
        OnToken: func(t string) { fmt.Print(t) },
    },
})
```

See [`example/main.go`](example/main.go) for a runnable sketch.

## Key types

- **`LLMClient`** — interface for communicating with any LLM backend
- **`Request`** / **`Response`** — provider-agnostic request and streamed response chunk
- **`RunConfig`** — bundles parameters for `Run()`, including client, request, tools, context strategy, and callbacks
- **`Message`** — a single message in a conversation
- **`ToolCall`** — an LLM's decision to invoke a tool
- **`Run()`** — executes the conversation loop: `Run(ctx, RunConfig) (*Result, error)`
- **`Stream()`** — executes the loop and returns a channel of `Event` values instead of using callbacks
- **`Event`** — streaming event emitted by `Stream()` (token, thinking, tool use, trim, done, or error)
- **`ContextStrategy`** — interface for trimming conversation history before each LLM call (implementations: `SlidingWindow`, `TokenBudget`, `TokenBudgetWithMinWindow`)
- **`Callbacks`** — optional hooks for tokens, thinking, tool use, context trimming, and completion

## License

MIT — see [LICENSE](LICENSE).
