package gossip

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/user/rate-limiter/internal/limiter"
)

// Node represents a participant in the rate limiter cluster.
type Node struct {
	ml       *memberlist.Memberlist
	delegate *rateLimiterDelegate
}

// NewNode initializes a new SWIM gossip node.
func NewNode(bindAddr string, bindPort int, store *limiter.Store) (*Node, error) {
	config := memberlist.DefaultLANConfig()
	config.BindAddr = bindAddr
	config.BindPort = bindPort
	config.Name = fmt.Sprintf("%s:%d", bindAddr, bindPort)

	d := newDelegate(store)
	config.Delegate = d

	ml, err := memberlist.Create(config)
	if err != nil {
		return nil, err
	}

	d.broadcasts.NumNodes = func() int {
		return ml.NumMembers()
	}

	return &Node{
		ml:       ml,
		delegate: d,
	}, nil
}

// Join connects this node to an existing cluster.
func (n *Node) Join(knownNodes []string) error {
	if len(knownNodes) > 0 {
		_, err := n.ml.Join(knownNodes)
		if err != nil {
			return err
		}
		log.Printf("Node %s joined cluster. Total members: %d", n.ml.LocalNode().Name, n.ml.NumMembers())
	}
	return nil
}

// Leave gracefully exits the cluster.
func (n *Node) Leave() error {
	return n.ml.Leave(time.Second * 5)
}

// BroadcastConsumed gossips the consumed tokens to the cluster.
func (n *Node) BroadcastConsumed(clientID string, consumed uint64) {
	n.delegate.AddDelta(clientID, consumed)
}

// NumMembers returns the current perceived cluster size.
func (n *Node) NumMembers() int {
	return n.ml.NumMembers()
}
