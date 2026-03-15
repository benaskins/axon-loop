package loop

import "log/slog"

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
