# Deployment & Integration Testing

The Rate Limiter is built to run natively compiled or inside Docker containers. Because the system utilizes stateful Gossip UDP networking, orchestrating tests and deployment requires careful port mapping and network privileges.

## Building and Running a Node

When running outside of Docker, the `main` entrypoint exposes CLI flags to configure the Gossip node and API endpoints.

```bash
# Build the binary
go build -o rate-limiter ./cmd/limiter

# Start the seed node (Node 1)
./rate-limiter -bind 127.0.0.1 -port 8001 -grpc :50051 -udp :6001

# Start Node 2 and join the cluster via Node 1
./rate-limiter -bind 127.0.0.1 -port 8002 -grpc :50052 -udp :6002 -join 127.0.0.1:8001
```

## Docker Compose 5-Node Cluster

We bundle a `docker-compose.yml` to spin up five identical gateway nodes. This setup is crucial for evaluating network partition (Jepsen-style) behaviors locally. 

Because `memberlist` uses port `7946` for SWIM traffic, and our custom UDP API uses `6000`, the compose file sets up a shared bridge network `rlnet`.

```bash
docker compose up -d --build
```
This boots `node1`, `node2`, `node3`, `node4`, and `node5`.
- `node1` acts as the initial state seed.
- Nodes 2-5 will auto-join `node1:7946`. If `node1` falls offline, Nodes 2-5 continue gossiping with each other seamlessly.

## Jepsen-Style Network Partitions

To prove the decentralized nature of the core engine, the repository ships with `scripts/jepsen_test.sh`.

### What it does:
1. Orchestrates the 5-node cluster using Docker Compose.
2. Uses Docker `exec` to inject `iptables` DROP rules directly into the kernel network stack of `node3`.
```bash
# Simulates a complete network partition
iptables -A INPUT -p udp --dport 7946 -j DROP
```
3. During this partition, `node3`'s gossip listener goes dark. It stops receiving tokens drained by `node1, 2, 4, 5`.
4. However, `node3` **does not crash or hang**. Its memory-local `TokenBucket` continues to function. If Node 3 serves an API gateway, that gateway continues responding to traffic within its stale local limits constraint (i.e. "fail open" or "fail gracefully" paradigm).

Once the partition heals (the iptables rules are removed), Node 3 resumes listening to SWIM `memberlist` broadcasts, and its local buckets eventually reconcile.

## CI/CD Pipeline Integration

Our `.github/workflows/ci.yml` strictly enforces the reliability of the system on every Pull Request to `main`.
1. It downloads Go `1.24` and triggers `go test -v -race -bench=. ./...` 
2. It asserts the core lock-free atomics do not trigger race conditions in Go's strict mode.
3. It spins up the `docker-compose` cluster and runs the Jepsen test natively.
