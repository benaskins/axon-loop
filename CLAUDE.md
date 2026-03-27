@AGENTS.md

## Conventions
- `RunConfig` bundles all conversation parameters — do not add loose function arguments
- Use callbacks for streaming events, not channels
- `ContextStrategy` interface (`SlidingWindow`, `TokenBudget`) handles message trimming
- `LLMClient` is the key abstraction — all provider interaction goes through it

## Constraints
- Never import `github.com/benaskins/axon` — this is a pure conversation library, no HTTP
- Depends only on axon-tool (for tool types); do not add other axon-* dependencies
- Do not leak provider-specific details (model names, token formats) into public types
- `Event` types are the streaming contract — keep them provider-agnostic

## Testing
- `go test ./...` — tests use mock `LLMClient` implementations, no live providers
- `go vet ./...` for lint
