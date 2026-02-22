package server

import (
	"log"
	"net"

	"github.com/user/rate-limiter/internal/gossip"
	"github.com/user/rate-limiter/internal/limiter"
)

// StartUDPServer starts an ultra-low-latency UDP server for rate-limit checks.
// It reads simply the client ID as a string and responds with 1 byte (1 for logic allowed, 0 for denied)
func StartUDPServer(addr string, store *limiter.Store, node *gossip.Node) (*net.UDPConn, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, err
	}

	go func() {
		buf := make([]byte, 1024)
		for {
			n, peer, err := conn.ReadFromUDP(buf)
			if err != nil {
				log.Printf("UDP server read error: %v", err)
				continue
			}

			if n == 0 {
				continue
			}

			clientID := string(buf[:n])
			bucket := store.GetOrCreate(clientID, 1000, 100)
			allowed := bucket.Allow()

			var resp []byte
			if allowed {
				resp = []byte{1}
				node.BroadcastConsumed(clientID, 1)
			} else {
				resp = []byte{0}
			}

			conn.WriteToUDP(resp, peer)
		}
	}()

	return conn, nil
}
