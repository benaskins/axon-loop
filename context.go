package loop

import "log/slog"

// ContextStrategy controls how conversation history is trimmed before
// each LLM request. Implementations receive the full message slice and
// return the subset to send. The system prompt (if present as the first
// message) should generally be preserved.
type ContextStrategy interface {
	// Trim returns the messages to send to the LLM.
	// The input slice must not be modified.
	Trim(messages []Message) []Message
}

// ContextStrategyFunc adapts a plain function into a ContextStrategy.
type ContextStrategyFunc func(messages []Message) []Message

func (f ContextStrategyFunc) Trim(messages []Message) []Message { return f(messages) }

// TokenBudget trims conversation history to fit within an estimated token
// budget. The system prompt is always preserved; older messages are dropped
// first. This is the strategy used when Request.MaxTokens is set without
// an explicit ContextStrategy.
func TokenBudget(budget int) ContextStrategy {
	return ContextStrategyFunc(func(messages []Message) []Message {
		return trimToTokenBudget(messages, budget)
	})
}

// TokenBudgetWithMinWindow trims to a token budget but guarantees at
// least minMessages conversation messages are kept (plus the system
// prompt). Whichever approach retains more messages wins. This prevents
// a large system prompt from starving the conversation history.
func TokenBudgetWithMinWindow(budget, minMessages int) ContextStrategy {
	tokenStrategy := TokenBudget(budget)
	windowStrategy := SlidingWindow(minMessages)
	return ContextStrategyFunc(func(messages []Message) []Message {
		byBudget := tokenStrategy.Trim(messages)
		byWindow := windowStrategy.Trim(messages)
		if len(byBudget) >= len(byWindow) {
			return byBudget
		}
		return byWindow
	})
}

// SlidingWindow keeps the system prompt plus the last n conversation
// messages. Useful when you want a fixed-size history regardless of
// token count.
func SlidingWindow(n int) ContextStrategy {
	return ContextStrategyFunc(func(messages []Message) []Message {
		if len(messages) == 0 {
			return messages
		}

		// Separate system prompt
		start := 0
		if messages[0].Role == RoleSystem {
			start = 1
		}

		conversation := messages[start:]
		if n <= 0 {
			// Keep only the system prompt (if any).
			if start == 1 {
				return []Message{messages[0]}
			}
			return nil
		}

		if len(conversation) <= n {
			return messages
		}

		trimmed := len(conversation) - n
		kept := conversation[trimmed:]

		slog.Info("context trimmed (sliding window)",
			"original_messages", len(messages),
			"kept_messages", len(kept)+start,
			"trimmed_messages", trimmed,
			"window_size", n,
		)

		if start == 1 {
			return append([]Message{messages[0]}, kept...)
		}
		return kept
	})
}

// droppedMessages computes which messages from original are absent in
// trimmed. It walks both slices in order, treating trimmed as a
// subsequence of original. Messages in original that don't appear in
// trimmed are returned in their original order.
func droppedMessages(original, trimmed []Message) []Message {
	if len(trimmed) >= len(original) {
		return nil
	}

	var dropped []Message
	j := 0
	for i := range original {
		if j < len(trimmed) && messageEqual(original[i], trimmed[j]) {
			j++
		} else {
			dropped = append(dropped, original[i])
		}
	}
	return dropped
}

// messageEqual reports whether two messages have the same role, content,
// and tool call ID. This is sufficient for identifying messages across
// trim boundaries where slice identity is lost.
func messageEqual(a, b Message) bool {
	return a.Role == b.Role && a.Content == b.Content && a.ToolCallID == b.ToolCallID
}

// estimateTokens returns a rough token count for a string.
// Uses the ~4 characters per token heuristic common across
// LLM tokenizers. Not precise, but good enough for budgeting.
func estimateTokens(s string) int {
	return (len(s) + 3) / 4
}

// messageTokens estimates the token count for a single message,
// including role overhead.
func messageTokens(m Message) int {
	tokens := 4 // role + framing overhead
	tokens += estimateTokens(m.Content)
	tokens += estimateTokens(m.Thinking)
	for _, tc := range m.ToolCalls {
		tokens += 10 // tool call overhead
		tokens += estimateTokens(tc.Name)
		for k, v := range tc.Arguments {
			tokens += estimateTokens(k)
			if s, ok := v.(string); ok {
				tokens += estimateTokens(s)
			} else {
				tokens += 5 // non-string arg estimate
			}
		}
	}
	return tokens
}

// trimToTokenBudget trims messages to fit within a token budget.
// The system prompt (first message if role == "system") is always
// preserved. Recent messages are kept; older messages are dropped.
// Returns the trimmed slice.
func trimToTokenBudget(messages []Message, budget int) []Message {
	if budget <= 0 || len(messages) == 0 {
		return messages
	}

	// Separate system prompt from conversation
	var system *Message
	conversation := messages
	if messages[0].Role == "system" {
		system = &messages[0]
		conversation = messages[1:]
	}

	// Calculate system prompt cost
	systemCost := 0
	if system != nil {
		systemCost = messageTokens(*system)
	}

	remaining := budget - systemCost
	if remaining <= 0 {
		// System prompt alone exceeds budget — send it anyway
		if system != nil {
			return []Message{*system}
		}
		return messages
	}

	// Walk backwards from most recent, accumulating until budget exhausted
	kept := make([]Message, 0, len(conversation))
	used := 0
	for i := len(conversation) - 1; i >= 0; i-- {
		cost := messageTokens(conversation[i])
		if used+cost > remaining {
			break
		}
		kept = append(kept, conversation[i])
		used += cost
	}

	// Reverse to restore chronological order
	for i, j := 0, len(kept)-1; i < j; i, j = i+1, j-1 {
		kept[i], kept[j] = kept[j], kept[i]
	}

	trimmed := len(conversation) - len(kept)
	if trimmed > 0 {
		slog.Info("context trimmed",
			"original_messages", len(messages),
			"kept_messages", len(kept)+1, // +1 for system
			"trimmed_messages", trimmed,
			"budget_tokens", budget,
			"used_tokens", used+systemCost,
		)
	}

	// Reassemble with system prompt
	if system != nil {
		return append([]Message{*system}, kept...)
	}
	return kept
}
