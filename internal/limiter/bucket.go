package limiter

import (
	"sync/atomic"
	"time"
)

// TokenBucket implements a lock-free token bucket using a single 64-bit atomic integer.
// The state packs: 
//   - 44 bits: Timestamp in milliseconds (wraps after ~550 years from epoch).
//   - 20 bits: Available tokens (max 1,048,575 tokens).
//
// This ensures atomic updates without lock contention, crucial for sub-millisecond p99 latency.
type TokenBucket struct {
	state      atomic.Uint64
	capacity   uint64
	ratePerSec uint64
}

const (
	tokensMask uint64 = 0xFFFFF // 20 bits
	timeShift  uint64 = 20
)

// NewTokenBucket creates a new lock-free token bucket.
func NewTokenBucket(capacity, ratePerSec uint64) *TokenBucket {
	if capacity > tokensMask {
		capacity = tokensMask
	}

	tb := &TokenBucket{
		capacity:   capacity,
		ratePerSec: ratePerSec,
	}

	nowMs := uint64(time.Now().UnixMilli())
	initialState := (nowMs << timeShift) | capacity
	tb.state.Store(initialState)

	return tb
}

// Allow is a convenience method for AllowN(1)
func (tb *TokenBucket) Allow() bool {
	return tb.AllowN(1)
}

// AllowN attempts to consume N tokens from the bucket.
// Returns true if tokens are available, otherwise false.
func (tb *TokenBucket) AllowN(n uint64) bool {
	if n > tb.capacity {
		return false
	}

	for {
		state := tb.state.Load()
		lastTimeMs := state >> timeShift
		tokens := state & tokensMask

		nowMs := uint64(time.Now().UnixMilli())
		if nowMs < lastTimeMs {
			// Handle clock skew / backward jumps gracefully
			nowMs = lastTimeMs 
		}

		elapsedMs := nowMs - lastTimeMs

		// Calculate tokens to add. ratePerSec is tokens/second.
		// tokens/ms = ratePerSec / 1000
		tokensToAdd := (elapsedMs * tb.ratePerSec) / 1000

		newTokens := tokens + tokensToAdd
		var newTimeMs uint64

		if newTokens >= tb.capacity {
			newTokens = tb.capacity
			newTimeMs = nowMs // The bucket is full, advance last refilled time to strictly now.
		} else if tokensToAdd > 0 {
			// We partially filled. We only advance the time by the exact milliseconds
			// representing the tokens we added to avoid losing fractional tokens.
			msUsedForTokens := (tokensToAdd * 1000) / tb.ratePerSec
			newTimeMs = lastTimeMs + msUsedForTokens
		} else {
			// No tokens added, keep previous time to accumulate elapsed time
			newTimeMs = lastTimeMs
		}

		if newTokens < n {
			return false // Not enough tokens
		}

		newTokens -= n
		newState := (newTimeMs << timeShift) | (newTokens & tokensMask)

		// CAS the new state back. If another goroutine modified it, loop and retry.
		if tb.state.CompareAndSwap(state, newState) {
			return true
		}
	}
}
