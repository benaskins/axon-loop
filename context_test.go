package loop

import "testing"

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"hi", 1},       // 2 chars -> 1 token
		{"hello", 2},    // 5 chars -> 2 tokens
		{"hello world how are you doing today", 9}, // 35 chars -> 9 tokens
	}

	for _, tt := range tests {
		got := estimateTokens(tt.input)
		if got != tt.want {
			t.Errorf("estimateTokens(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestTrimToTokenBudget_NoBudget(t *testing.T) {
	msgs := []Message{
		{Role: "system", Content: "you are helpful"},
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi there"},
	}

	result := trimToTokenBudget(msgs, 0)
	if len(result) != 3 {
		t.Errorf("expected 3 messages with no budget, got %d", len(result))
	}
}

func TestTrimToTokenBudget_PreservesSystemPrompt(t *testing.T) {
	msgs := []Message{
		{Role: "system", Content: "you are helpful"},
		{Role: "user", Content: "message 1"},
		{Role: "assistant", Content: "response 1"},
		{Role: "user", Content: "message 2"},
		{Role: "assistant", Content: "response 2"},
	}

	// Very small budget — should keep system + most recent
	result := trimToTokenBudget(msgs, 30)

	if len(result) < 2 {
		t.Fatalf("expected at least system + 1 message, got %d", len(result))
	}
	if result[0].Role != "system" {
		t.Error("first message should be system prompt")
	}
	// Last message should be the most recent
	if result[len(result)-1].Content != "response 2" {
		t.Errorf("last message = %q, want most recent", result[len(result)-1].Content)
	}
}

func TestTrimToTokenBudget_KeepsAllWhenUnderBudget(t *testing.T) {
	msgs := []Message{
		{Role: "system", Content: "hi"},
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hey"},
	}

	result := trimToTokenBudget(msgs, 100000)
	if len(result) != 3 {
		t.Errorf("expected all 3 messages under large budget, got %d", len(result))
	}
}

func TestTrimToTokenBudget_NoSystemPrompt(t *testing.T) {
	msgs := []Message{
		{Role: "user", Content: "first"},
		{Role: "assistant", Content: "second"},
		{Role: "user", Content: "third"},
	}

	// Small budget — should keep most recent
	result := trimToTokenBudget(msgs, 20)

	if len(result) == 0 {
		t.Fatal("expected at least one message")
	}
	if result[len(result)-1].Content != "third" {
		t.Errorf("last message = %q, want most recent", result[len(result)-1].Content)
	}
}

func TestTrimToTokenBudget_TrimsOldestFirst(t *testing.T) {
	msgs := []Message{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: "old message that should be dropped"},
		{Role: "assistant", Content: "old response that should be dropped"},
		{Role: "user", Content: "recent"},
		{Role: "assistant", Content: "latest"},
	}

	// Budget enough for system + last 2 messages but not all 4
	systemCost := messageTokens(msgs[0])
	recentCost := messageTokens(msgs[3]) + messageTokens(msgs[4])
	budget := systemCost + recentCost + 5 // tight budget

	result := trimToTokenBudget(msgs, budget)

	if result[0].Role != "system" {
		t.Error("system prompt should be preserved")
	}
	if len(result) > 3 {
		t.Errorf("expected at most 3 messages (sys + 2 recent), got %d", len(result))
	}

	// Should contain the recent messages, not the old ones
	hasOld := false
	hasRecent := false
	for _, m := range result {
		if m.Content == "old message that should be dropped" {
			hasOld = true
		}
		if m.Content == "latest" {
			hasRecent = true
		}
	}
	if hasOld {
		t.Error("old messages should have been trimmed")
	}
	if !hasRecent {
		t.Error("recent messages should be kept")
	}
}

func TestMessageTokens_IncludesToolCalls(t *testing.T) {
	m := Message{
		Role:    "assistant",
		Content: "",
		ToolCalls: []ToolCall{
			{Name: "read_file", Arguments: map[string]any{"path": "/some/file.go"}},
		},
	}

	tokens := messageTokens(m)
	if tokens < 10 {
		t.Errorf("expected tool call message to have >10 tokens, got %d", tokens)
	}
}

// --- ContextStrategy tests ---

func TestTokenBudget_MatchesTrimToTokenBudget(t *testing.T) {
	msgs := []Message{
		{Role: "system", Content: "you are helpful"},
		{Role: "user", Content: "old message"},
		{Role: "assistant", Content: "old response"},
		{Role: "user", Content: "recent"},
		{Role: "assistant", Content: "latest"},
	}

	budget := 30
	direct := trimToTokenBudget(msgs, budget)
	strategy := TokenBudget(budget).Trim(msgs)

	if len(direct) != len(strategy) {
		t.Fatalf("TokenBudget strategy returned %d messages, trimToTokenBudget returned %d", len(strategy), len(direct))
	}
	for i := range direct {
		if direct[i].Content != strategy[i].Content {
			t.Errorf("message %d: direct=%q, strategy=%q", i, direct[i].Content, strategy[i].Content)
		}
	}
}

func TestSlidingWindow_KeepsLastN(t *testing.T) {
	msgs := []Message{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: "msg1"},
		{Role: "assistant", Content: "resp1"},
		{Role: "user", Content: "msg2"},
		{Role: "assistant", Content: "resp2"},
		{Role: "user", Content: "msg3"},
	}

	result := SlidingWindow(2).Trim(msgs)

	if len(result) != 3 { // system + last 2
		t.Fatalf("expected 3 messages, got %d", len(result))
	}
	if result[0].Role != "system" {
		t.Error("first message should be system prompt")
	}
	if result[1].Content != "resp2" {
		t.Errorf("expected resp2, got %q", result[1].Content)
	}
	if result[2].Content != "msg3" {
		t.Errorf("expected msg3, got %q", result[2].Content)
	}
}

func TestSlidingWindow_KeepsAllWhenUnderWindow(t *testing.T) {
	msgs := []Message{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: "hello"},
	}

	result := SlidingWindow(10).Trim(msgs)
	if len(result) != 2 {
		t.Errorf("expected all 2 messages, got %d", len(result))
	}
}

func TestSlidingWindow_NoSystemPrompt(t *testing.T) {
	msgs := []Message{
		{Role: "user", Content: "msg1"},
		{Role: "assistant", Content: "resp1"},
		{Role: "user", Content: "msg2"},
		{Role: "assistant", Content: "resp2"},
	}

	result := SlidingWindow(2).Trim(msgs)

	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(result))
	}
	if result[0].Content != "msg2" {
		t.Errorf("expected msg2, got %q", result[0].Content)
	}
	if result[1].Content != "resp2" {
		t.Errorf("expected resp2, got %q", result[1].Content)
	}
}

func TestSlidingWindow_ZeroKeepsOnlySystem(t *testing.T) {
	msgs := []Message{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: "msg1"},
		{Role: "assistant", Content: "resp1"},
	}

	result := SlidingWindow(0).Trim(msgs)

	if len(result) != 1 {
		t.Fatalf("expected 1 message (system only), got %d", len(result))
	}
	if result[0].Role != "system" {
		t.Errorf("expected system prompt, got %q", result[0].Role)
	}
}

func TestSlidingWindow_ZeroNoSystem(t *testing.T) {
	msgs := []Message{
		{Role: "user", Content: "msg1"},
		{Role: "assistant", Content: "resp1"},
	}

	result := SlidingWindow(0).Trim(msgs)

	if len(result) != 0 {
		t.Fatalf("expected 0 messages, got %d", len(result))
	}
}

func TestSlidingWindow_EmptyMessages(t *testing.T) {
	result := SlidingWindow(5).Trim(nil)
	if len(result) != 0 {
		t.Errorf("expected 0 messages for nil input, got %d", len(result))
	}
}

func TestContextStrategyFunc(t *testing.T) {
	called := false
	strategy := ContextStrategyFunc(func(msgs []Message) []Message {
		called = true
		return msgs
	})

	msgs := []Message{{Role: "user", Content: "hello"}}
	result := strategy.Trim(msgs)

	if !called {
		t.Error("function was not called")
	}
	if len(result) != 1 {
		t.Errorf("expected 1 message, got %d", len(result))
	}
}

func TestDroppedMessages_WithSystemPrompt(t *testing.T) {
	original := []Message{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: "old1"},
		{Role: "assistant", Content: "old2"},
		{Role: "user", Content: "recent"},
	}
	trimmed := []Message{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: "recent"},
	}

	dropped := droppedMessages(original, trimmed)
	if len(dropped) != 2 {
		t.Fatalf("expected 2 dropped, got %d", len(dropped))
	}
	if dropped[0].Content != "old1" {
		t.Errorf("dropped[0] = %q, want old1", dropped[0].Content)
	}
	if dropped[1].Content != "old2" {
		t.Errorf("dropped[1] = %q, want old2", dropped[1].Content)
	}
}

func TestDroppedMessages_NoTrimming(t *testing.T) {
	msgs := []Message{
		{Role: "user", Content: "hello"},
	}
	dropped := droppedMessages(msgs, msgs)
	if len(dropped) != 0 {
		t.Errorf("expected 0 dropped when no trimming, got %d", len(dropped))
	}
}

func TestDroppedMessages_MiddleDrop(t *testing.T) {
	// A custom strategy that drops from the middle, not just a prefix.
	original := []Message{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: "keep1"},
		{Role: "assistant", Content: "drop-me"},
		{Role: "user", Content: "keep2"},
		{Role: "assistant", Content: "keep3"},
	}
	trimmed := []Message{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: "keep1"},
		{Role: "user", Content: "keep2"},
		{Role: "assistant", Content: "keep3"},
	}

	dropped := droppedMessages(original, trimmed)
	if len(dropped) != 1 {
		t.Fatalf("expected 1 dropped, got %d", len(dropped))
	}
	if dropped[0].Content != "drop-me" {
		t.Errorf("dropped[0] = %q, want drop-me", dropped[0].Content)
	}
}

func TestDroppedMessages_MultipleNonContiguous(t *testing.T) {
	original := []Message{
		{Role: "user", Content: "a"},
		{Role: "assistant", Content: "b"},
		{Role: "user", Content: "c"},
		{Role: "assistant", Content: "d"},
		{Role: "user", Content: "e"},
	}
	// Strategy keeps a, c, e — drops b and d.
	trimmed := []Message{
		{Role: "user", Content: "a"},
		{Role: "user", Content: "c"},
		{Role: "user", Content: "e"},
	}

	dropped := droppedMessages(original, trimmed)
	if len(dropped) != 2 {
		t.Fatalf("expected 2 dropped, got %d", len(dropped))
	}
	if dropped[0].Content != "b" {
		t.Errorf("dropped[0] = %q, want b", dropped[0].Content)
	}
	if dropped[1].Content != "d" {
		t.Errorf("dropped[1] = %q, want d", dropped[1].Content)
	}
}
