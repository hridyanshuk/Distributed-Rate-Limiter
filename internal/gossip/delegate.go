package gossip

import (
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/user/rate-limiter/internal/limiter"
	"github.com/user/rate-limiter/internal/netutil"
)

type broadcastMsg struct {
	msg []byte
}

func (b *broadcastMsg) Invalidates(other memberlist.Broadcast) bool { return false }
func (b *broadcastMsg) Message() []byte                             { return b.msg }
func (b *broadcastMsg) Finished()                                   {}

// rateLimiterDelegate implements memberlist.Delegate
type rateLimiterDelegate struct {
	store      *limiter.Store
	broadcasts *memberlist.TransmitLimitedQueue
	seqID      uint32
	seenSeqs   map[uint32]time.Time
	mu         sync.Mutex
}

func newDelegate(store *limiter.Store) *rateLimiterDelegate {
	// Initialize seqID to a random value to avoid cross-node collisions in tests
	initSeq := rand.Uint32()

	d := &rateLimiterDelegate{
		store:    store,
		seqID:    initSeq,
		seenSeqs: make(map[uint32]time.Time),
	}
	d.broadcasts = &memberlist.TransmitLimitedQueue{
		NumNodes:       func() int { return 1 }, // Updated when memberlist starts
		RetransmitMult: 3,
	}
	return d
}

// AddDelta queues a delta message for reliable gossip broadcast.
func (d *rateLimiterDelegate) AddDelta(clientID string, consumed uint64) {
	d.mu.Lock()
	d.seqID++
	mySeq := d.seqID
	d.mu.Unlock()

	deltas := []netutil.Delta{{ClientID: clientID, Consumed: consumed}}
	
	buf := make([]byte, 1024)
	n, err := netutil.EncodeDeltaMessage(buf, mySeq, deltas)
	if err != nil {
		log.Printf("Failed to encode delta: %v", err)
		return
	}

	d.broadcasts.QueueBroadcast(&broadcastMsg{msg: buf[:n]})
}

func (d *rateLimiterDelegate) NodeMeta(limit int) []byte { return []byte{} }

func (d *rateLimiterDelegate) NotifyMsg(buf []byte) {
	seqID, deltas, err := netutil.DecodeDeltaMessage(buf)
	if err != nil {
		return
	}

	d.mu.Lock()
	if _, seen := d.seenSeqs[seqID]; seen {
		d.mu.Unlock()
		return
	}
	d.seenSeqs[seqID] = time.Now()
	
	// Basic prune to prevent memory leak
	if len(d.seenSeqs) > 1000 {
	    d.seenSeqs = make(map[uint32]time.Time)
	}
	d.mu.Unlock()

	for _, delta := range deltas {
		// Using standard token bucket settings (1000 cap, 100 rate)
		bucket := d.store.GetOrCreate(delta.ClientID, 1000, 100)
		bucket.AllowN(delta.Consumed) // Drain remote consumptions
	}
}

func (d *rateLimiterDelegate) GetBroadcasts(overhead, limit int) [][]byte {
	return d.broadcasts.GetBroadcasts(overhead, limit)
}

func (d *rateLimiterDelegate) LocalState(join bool) []byte {
	return []byte{}
}

func (d *rateLimiterDelegate) MergeRemoteState(buf []byte, join bool) {}
