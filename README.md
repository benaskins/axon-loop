# axon-loop

A provider-agnostic conversation loop for LLM-powered agents. Part of [lamina](https://github.com/benaskins/lamina) — each axon package can be used independently.

Handles message exchange, tool call dispatch, and streaming — with no HTTP, persistence, or UI concerns.

## Install

```
go get github.com/benaskins/axon-loop@latest
```

Requires Go 1.24+.

## Usage

Implement the `LLMClient` interface for your LLM backend, define tools, and run:

```go
result, err := loop.Run(ctx, loop.RunConfig{
    Client:   myLLMClient,
    Messages: messages,
    Tools:    toolDefs,
    Callbacks: loop.Callbacks{
        OnToken: func(token string) { fmt.Print(token) },
    },
})
```

### Key types

- `LLMClient` — interface abstracting communication with any LLM backend
- `Request` / `Response` — provider-agnostic request and streamed response
- `Message` — a single message in a conversation
- `ToolCall` — an LLM's decision to invoke a tool
- `Run()` — executes the agent conversation loop with tool dispatch

## License

MIT — see [LICENSE](LICENSE).
