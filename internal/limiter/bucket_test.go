package limiter

import (
	"testing"
	"time"
)

func TestTokenBucket_LockFree(t *testing.T) {
	// 10 capacity, 10 tokens per second (1 token / 100ms)
	tb := NewTokenBucket(10, 10)

	// Consume all 10 tokens immediately
	for i := 0; i < 10; i++ {
		if !tb.Allow() {
			t.Fatalf("expected allow on iteration %d", i)
		}
	}

	// 11th token should be denied
	if tb.Allow() {
		t.Fatal("expected deny, bucket should be empty")
	}

	// Wait 150ms to allow 1 token to refill (100ms per token)
	time.Sleep(150 * time.Millisecond)

	if !tb.Allow() {
		t.Fatal("expected allow after sleep (1 token should have refilled)")
	}
	
	if tb.Allow() {
		t.Fatal("expected deny after 1 allowed (bucket should be empty again)")
	}
}

// Benchmark the lock-free token bucket's AllowN method
// using high concurrency to ensure no lock contention problems.
func BenchmarkTokenBucket_AllowParallel(b *testing.B) {
	tb := NewTokenBucket(1_000_000, 1_000_000)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			tb.Allow()
		}
	})
}
