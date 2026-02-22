package limiter

import (
	"hash/fnv"
	"sync"
)

const numShards = 256

// Store is a highly concurrent, sharded hash map for storing token buckets.
// Sharding drastically reduces lock contention across different clients.
type Store struct {
	shards [numShards]*shard
}

type shard struct {
	mu      sync.RWMutex
	buckets map[string]*TokenBucket
}

// NewStore initializes a new sharded bucket store.
func NewStore() *Store {
	s := &Store{}
	for i := 0; i < numShards; i++ {
		s.shards[i] = &shard{
			buckets: make(map[string]*TokenBucket),
		}
	}
	return s
}

// getShard returns the specific shard for a given key using FNV-1a hashing.
func (s *Store) getShard(key string) *shard {
	h := fnv.New32a()
	h.Write([]byte(key))
	return s.shards[h.Sum32()%numShards]
}

// GetOrCreate retrieves an existing TokenBucket for a client key, or creates one if it doesn't exist.
func (s *Store) GetOrCreate(key string, capacity, ratePerSec uint64) *TokenBucket {
	shard := s.getShard(key)

	// Fast path: read lock
	shard.mu.RLock()
	b, exists := shard.buckets[key]
	shard.mu.RUnlock()

	if exists {
		return b
	}

	// Slow path: write lock
	shard.mu.Lock()
	defer shard.mu.Unlock()

	// Double-check after acquiring write lock
	if b, exists := shard.buckets[key]; exists {
		return b
	}

	b = NewTokenBucket(capacity, ratePerSec)
	shard.buckets[key] = b
	return b
}
