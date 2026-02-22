package server

import (
	"context"
	"net"

	"google.golang.org/grpc"
	ratelimitpb "github.com/user/rate-limiter/api/v1"
	"github.com/user/rate-limiter/internal/gossip"
	"github.com/user/rate-limiter/internal/limiter"
)

type GRPCServer struct {
	ratelimitpb.UnimplementedRateLimiterServer
	store *limiter.Store
	node  *gossip.Node
}

func NewGRPCServer(store *limiter.Store, node *gossip.Node) *GRPCServer {
	return &GRPCServer{
		store: store,
		node:  node,
	}
}

func (s *GRPCServer) AllowRequest(ctx context.Context, req *ratelimitpb.AllowRequestArgs) (*ratelimitpb.AllowRequestResponse, error) {
	// Let's assume standard capacity 1000 and refill rate 100 for all clients.
	// In production, these might be read from a configuration based on the tier.
	bucket := s.store.GetOrCreate(req.ClientId, 1000, 100)
	allowed := bucket.Allow()

	if allowed {
		// Broadcast token consumption to the rest of the cluster via gossip
		s.node.BroadcastConsumed(req.ClientId, 1)
	}

	return &ratelimitpb.AllowRequestResponse{
		Allowed: allowed,
	}, nil
}

// StartGRPCServer initializes the gRPC service and begins listening.
func StartGRPCServer(addr string, store *limiter.Store, node *gossip.Node) (*grpc.Server, error) {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	grpcServer := grpc.NewServer()
	ratelimitpb.RegisterRateLimiterServer(grpcServer, NewGRPCServer(store, node))

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			panic(err) // For this prototype, panic on serve failure is fine
		}
	}()

	return grpcServer, nil
}
