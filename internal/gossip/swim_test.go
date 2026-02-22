package gossip

import (
	"log"
	"testing"
	"time"

	"github.com/user/rate-limiter/internal/limiter"
)

func init() {
	// Silence memberlist logging for tests
	log.SetFlags(0)
	log.SetOutput(new(DevNull))
}

type DevNull struct{}
func (DevNull) Write(p []byte) (int, error) { return len(p), nil }

func TestGossipStateSync(t *testing.T) {
	// Start node 1
	store1 := limiter.NewStore()
	node1, err := NewNode("127.0.0.1", 8001, store1)
	if err != nil {
		t.Fatalf("failed to start node1: %v", err)
	}
	defer node1.Leave()

	// Start node 2
	store2 := limiter.NewStore()
	node2, err := NewNode("127.0.0.1", 8002, store2)
	if err != nil {
		t.Fatalf("failed to start node2: %v", err)
	}
	defer node2.Leave()

	// Join node 2 to node 1
	err = node2.Join([]string{"127.0.0.1:8001"})
	if err != nil {
		t.Fatalf("failed to join node2 to node1: %v", err)
	}

	// Wait a bit for gossip to settle
	time.Sleep(500 * time.Millisecond)

	if node1.NumMembers() != 2 || node2.NumMembers() != 2 {
		t.Fatalf("expected 2 members, got %d and %d", node1.NumMembers(), node2.NumMembers())
	}

	// Node 1 consumes 500 tokens for clientX
	node1.BroadcastConsumed("clientX", 500)

	// Wait for local gossip propagation
	time.Sleep(1 * time.Second)

	// Verify node 2 received the delta
	// Node 2's bucket for clientX should have 500 tokens less than max (1000 - 500 = 500)
	// We check this by trying to consume 501 tokens on Node 2. It should fail.
	// We can check available tokens using a debug helper
	// But let's just use AllowN
	b2 := store2.GetOrCreate("clientX", 1000, 100)
	
	// Print b2 state
	// To do this we can try to allow 1 by 1 until it fails
	count := 0
	for b2.Allow() {
		count++
	}
	t.Logf("Tokens available: %d", count)
	
	if count >= 1000 {
		t.Fatalf("expected node2 to have < 1000 tokens due to gossip, got %d", count)
	}

	if count < 400 {
		t.Fatalf("expected node2 to have at least 400 tokens, got %d", count)
	}
}
