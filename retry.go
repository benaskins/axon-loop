package loop

import (
	"context"
	"errors"
	"math"
	"math/rand/v2"
	"time"

	talk "github.com/benaskins/axon-talk"
)

type retryConfig struct {
	maxRetries int
	initial    time.Duration
	max        time.Duration
	retryable  func(error) bool
}

// RetryOption configures the retry decorator.
type RetryOption func(*retryConfig)

// WithMaxRetries sets the maximum number of retry attempts. Default is 3.
func WithMaxRetries(n int) RetryOption {
	return func(c *retryConfig) { c.maxRetries = n }
}

// WithBackoff sets the initial and maximum backoff durations. Default is
// 1s initial, 30s max. Jittered exponential backoff is applied.
func WithBackoff(initial, max time.Duration) RetryOption {
	return func(c *retryConfig) { c.initial = initial; c.max = max }
}

// WithRetryable sets a custom function to determine if an error is retryable.
// By default, errors with HTTP status 429, 500, 502, or 503 are retried.
func WithRetryable(fn func(error) bool) RetryOption {
	return func(c *retryConfig) { c.retryable = fn }
}

// WithRetry wraps an LLMClient with retry logic. Requests that fail with
// retryable errors are retried with jittered exponential backoff.
//
// Streaming safety: if the callback fn has been invoked (tokens delivered),
// the request is not retried to avoid duplicate content.
func WithRetry(client talk.LLMClient, opts ...RetryOption) talk.LLMClient {
	cfg := retryConfig{
		maxRetries: 3,
		initial:    1 * time.Second,
		max:        30 * time.Second,
		retryable:  defaultRetryable,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return &retryClient{inner: client, cfg: cfg}
}

type retryClient struct {
	inner talk.LLMClient
	cfg   retryConfig
}

func (r *retryClient) Chat(ctx context.Context, req *talk.Request, fn func(talk.Response) error) error {
	var lastErr error

	for attempt := range r.cfg.maxRetries + 1 {
		if attempt > 0 {
			delay := backoff(r.cfg.initial, r.cfg.max, attempt-1)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		called := false
		wrappedFn := func(resp talk.Response) error {
			called = true
			return fn(resp)
		}

		err := r.inner.Chat(ctx, req, wrappedFn)
		if err == nil {
			return nil
		}

		// Do not retry if the callback was already invoked (tokens delivered).
		if called {
			return err
		}

		if !r.cfg.retryable(err) {
			return err
		}

		lastErr = err
	}

	return lastErr
}

func defaultRetryable(err error) bool {
	var se *talk.StatusError
	if !errors.As(err, &se) {
		return false
	}
	switch se.StatusCode {
	case 429, 500, 502, 503:
		return true
	}
	return false
}

// backoff calculates jittered exponential backoff for the given attempt.
func backoff(initial, max time.Duration, attempt int) time.Duration {
	d := time.Duration(float64(initial) * math.Pow(2, float64(attempt)))
	if d > max {
		d = max
	}
	// Add jitter: 0.5x to 1.0x of computed delay.
	jitter := 0.5 + rand.Float64()*0.5
	return time.Duration(float64(d) * jitter)
}
