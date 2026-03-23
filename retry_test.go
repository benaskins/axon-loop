package loop

import (
	"context"
	"errors"
	"testing"
	"time"

	talk "github.com/benaskins/axon-talk"
)

type fakeClient struct {
	calls   int
	results []error
}

func (f *fakeClient) Chat(_ context.Context, _ *talk.Request, fn func(talk.Response) error) error {
	i := f.calls
	f.calls++
	if i < len(f.results) {
		if f.results[i] != nil {
			return f.results[i]
		}
	}
	return fn(talk.Response{Content: "ok", Done: true})
}

func TestWithRetry_NoError(t *testing.T) {
	inner := &fakeClient{results: []error{nil}}
	client := WithRetry(inner, WithBackoff(time.Millisecond, time.Millisecond))

	var got talk.Response
	err := client.Chat(context.Background(), &talk.Request{}, func(resp talk.Response) error {
		got = resp
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Content != "ok" {
		t.Errorf("content = %q, want ok", got.Content)
	}
	if inner.calls != 1 {
		t.Errorf("calls = %d, want 1", inner.calls)
	}
}

func TestWithRetry_RetriesOnStatusError(t *testing.T) {
	inner := &fakeClient{results: []error{
		&talk.StatusError{StatusCode: 429, Body: "rate limited", Provider: "test"},
		&talk.StatusError{StatusCode: 503, Body: "unavailable", Provider: "test"},
		nil, // third attempt succeeds
	}}
	client := WithRetry(inner, WithBackoff(time.Millisecond, time.Millisecond))

	err := client.Chat(context.Background(), &talk.Request{}, func(resp talk.Response) error {
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inner.calls != 3 {
		t.Errorf("calls = %d, want 3", inner.calls)
	}
}

func TestWithRetry_DoesNotRetryNonRetryable(t *testing.T) {
	inner := &fakeClient{results: []error{
		&talk.StatusError{StatusCode: 401, Body: "unauthorized", Provider: "test"},
	}}
	client := WithRetry(inner, WithBackoff(time.Millisecond, time.Millisecond))

	err := client.Chat(context.Background(), &talk.Request{}, func(resp talk.Response) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if inner.calls != 1 {
		t.Errorf("calls = %d, want 1 (should not retry 401)", inner.calls)
	}
}

func TestWithRetry_DoesNotRetryPlainError(t *testing.T) {
	inner := &fakeClient{results: []error{
		errors.New("connection refused"),
	}}
	client := WithRetry(inner, WithBackoff(time.Millisecond, time.Millisecond))

	err := client.Chat(context.Background(), &talk.Request{}, func(resp talk.Response) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if inner.calls != 1 {
		t.Errorf("calls = %d, want 1 (plain errors not retried by default)", inner.calls)
	}
}

func TestWithRetry_ExhaustsRetries(t *testing.T) {
	retryableErr := &talk.StatusError{StatusCode: 500, Body: "server error", Provider: "test"}
	inner := &fakeClient{results: []error{
		retryableErr, retryableErr, retryableErr, retryableErr,
	}}
	client := WithRetry(inner,
		WithMaxRetries(2),
		WithBackoff(time.Millisecond, time.Millisecond),
	)

	err := client.Chat(context.Background(), &talk.Request{}, func(resp talk.Response) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	// 1 initial + 2 retries = 3 calls
	if inner.calls != 3 {
		t.Errorf("calls = %d, want 3", inner.calls)
	}
}

func TestWithRetry_DoesNotRetryAfterCallback(t *testing.T) {
	calls := 0
	inner := &callbackThenFailClient{calls: &calls}
	client := WithRetry(inner, WithBackoff(time.Millisecond, time.Millisecond))

	err := client.Chat(context.Background(), &talk.Request{}, func(resp talk.Response) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1 (should not retry after callback)", calls)
	}
}

func TestWithRetry_RespectsContextCancellation(t *testing.T) {
	retryableErr := &talk.StatusError{StatusCode: 429, Body: "rate limited", Provider: "test"}
	inner := &fakeClient{results: []error{
		retryableErr, retryableErr, retryableErr,
	}}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	client := WithRetry(inner,
		WithMaxRetries(5),
		WithBackoff(time.Second, time.Second),
	)

	err := client.Chat(ctx, &talk.Request{}, func(resp talk.Response) error {
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

func TestWithRetry_CustomRetryable(t *testing.T) {
	inner := &fakeClient{results: []error{
		errors.New("flaky network"),
		nil,
	}}
	client := WithRetry(inner,
		WithBackoff(time.Millisecond, time.Millisecond),
		WithRetryable(func(err error) bool { return true }), // retry everything
	)

	err := client.Chat(context.Background(), &talk.Request{}, func(resp talk.Response) error {
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inner.calls != 2 {
		t.Errorf("calls = %d, want 2", inner.calls)
	}
}

// callbackThenFailClient delivers a token then returns a retryable error.
type callbackThenFailClient struct {
	calls *int
}

func (c *callbackThenFailClient) Chat(_ context.Context, _ *talk.Request, fn func(talk.Response) error) error {
	*c.calls++
	fn(talk.Response{Content: "partial"})
	return &talk.StatusError{StatusCode: 500, Body: "mid-stream", Provider: "test"}
}

func TestBackoff_Bounds(t *testing.T) {
	for attempt := range 10 {
		d := backoff(time.Second, 30*time.Second, attempt)
		if d > 30*time.Second {
			t.Errorf("attempt %d: backoff %v exceeds max 30s", attempt, d)
		}
		if d <= 0 {
			t.Errorf("attempt %d: backoff %v should be positive", attempt, d)
		}
	}
}
