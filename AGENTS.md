# axon-loop

Provider-agnostic conversation loop for LLM-powered agents.

## Build & Test

```bash
go test ./...
go vet ./...
```

## Key Files

- `run.go` — main conversation loop execution (`Run`, `Stream`, `RunConfig`, `Event` types)
- `context.go` — conversation context management (`ContextStrategy`, `SlidingWindow`, `TokenBudget`)
- `agent.go` — agent definition and lifecycle
- `doc.go` — package documentation
