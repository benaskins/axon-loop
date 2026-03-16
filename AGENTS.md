# axon-loop

Provider-agnostic conversation loop for LLM-powered agents.

## Build & Test

```bash
go test ./...
go vet ./...
```

## Key Files

- `agent.go` — agent definition and lifecycle
- `context.go` — conversation context management
- `run.go` — main conversation loop execution
