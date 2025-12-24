package ratelimit

import (
	"testing"
	"time"
)

func TestTokenBucketAllowAndRefill(t *testing.T) {
	tb := NewTokenBucket(2, 1) // 2 capacity, 1 token/sec

	if !tb.Allow() || !tb.Allow() {
		t.Fatalf("expected first two requests to pass")
	}
	if tb.Allow() {
		t.Fatalf("expected third request to be limited")
	}

	time.Sleep(1100 * time.Millisecond) // allow refill of 1 token
	if !tb.Allow() {
		t.Fatalf("expected request to pass after refill")
	}
}
