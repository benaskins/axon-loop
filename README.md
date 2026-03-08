# axon-loop

> Primitives · Part of the [lamina](https://github.com/benaskins/lamina-mono) workspace

Provider-agnostic conversation loop for LLM-powered agents. Handles message exchange, tool call dispatch, and streaming with no HTTP, persistence, or UI concerns. Bring your own `LLMClient` implementation (e.g. Ollama, OpenAI, Anthropic) and axon-loop drives the send-stream-tool-repeat cycle.

## Getting started

```bash
go get github.com/benaskins/axon-loop@latest
```

Requires Go 1.24+.

```go
result, err := loop.Run(ctx, client, &loop.Request{
    Model:    "llama3",
    Messages: []loop.Message{{Role: "user", Content: "Hello"}},
    Stream:   true,
}, tools, nil, loop.Callbacks{
    OnToken: func(t string) { fmt.Print(t) },
})
```

See [`example/main.go`](example/main.go) for a runnable sketch.

## Key types

- **`LLMClient`** — interface for communicating with any LLM backend
- **`Request`** / **`Response`** — provider-agnostic request and streamed response chunk
- **`Message`** — a single message in a conversation
- **`ToolCall`** — an LLM's decision to invoke a tool
- **`Run()`** — executes the conversation loop with tool dispatch and streaming callbacks
- **`Callbacks`** — optional hooks for tokens, thinking, tool use, and completion

## License

MIT — see [LICENSE](LICENSE).
