package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/user/rate-limiter/cmd/server"
	"github.com/user/rate-limiter/internal/gossip"
	"github.com/user/rate-limiter/internal/limiter"
)

func main() {
	bindAddr := flag.String("bind", "0.0.0.0", "Gossip bind address")
	bindPort := flag.Int("port", 7946, "Gossip bind port")
	grpcPort := flag.String("grpc", ":50051", "gRPC listen address")
	udpPort := flag.String("udp", ":6000", "UDP listen address")
	join := flag.String("join", "", "Comma separated list of nodes to join")
	flag.Parse()

	log.Printf("Starting sub-millisecond distributed rate limiter...")
	store := limiter.NewStore()

	node, err := gossip.NewNode(*bindAddr, *bindPort, store)
	if err != nil {
		log.Fatalf("Failed to initialize gossip node: %v", err)
	}
	defer node.Leave()

	if *join != "" {
		nodes := strings.Split(*join, ",")
		if err := node.Join(nodes); err != nil {
			log.Fatalf("Failed to join cluster: %v", err)
		}
	}

	grpcSrv, err := server.StartGRPCServer(*grpcPort, store, node)
	if err != nil {
		log.Fatalf("Failed to start gRPC server: %v", err)
	}
	log.Printf("gRPC server listening on %s", *grpcPort)

	udpConn, err := server.StartUDPServer(*udpPort, store, node)
	if err != nil {
		log.Fatalf("Failed to start UDP server: %v", err)
	}
	log.Printf("UDP ultra-low-latency server listening on %s", *udpPort)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Println("Shutting down...")
	grpcSrv.GracefulStop()
	udpConn.Close()
}
