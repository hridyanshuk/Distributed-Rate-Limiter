# Sub-Millisecond Distributed Rate Limiter

Welcome to the documentation for the Sub-Millisecond Distributed Rate Limiter!

This system is built from the ground up for **extreme performance** and **FAANG-scale** API protections. It departs from the traditional centralized Redis bottleneck, moving to a fully decentralized model using lock-free data structures in Go, synchronized via a highly optimized SWIM Gossip protocol.

---

## 📚 Documentation Index

We have split the documentation into multiple focused pages to dive deep into the specific subsystems:

1. **[High-Level Architecture](docs/architecture.md)**
   Understand the paradigm shift from centralized (Redis) to decentralized (Gossip) rate limiting. Contains system sequence diagrams and cluster topology overviews.
2. **[Low-Level Design (LLD)](docs/low_level_design.md)**
   Dive deep into the `9 nanosecond` lock-free Token Bucket algorithm. Learn how we use bit-packed 64-bit atomic integers, highly parallel sharded maps, and zero-allocation UDP binary serialization.
3. **[API Presentation Layer](docs/api.md)**
   Information about the gRPC and Ultra-Low-Latency UDP interfaces exposed to API gateways.
4. **[Deployment & Testing](docs/deployment.md)**
   Instructions on running the `docker-compose` cluster, running benchmarks, and injecting Jepsen-style network partitions to prove fault tolerance.

---

## 🚀 Quick Start

If you just want to run the code and see the cluster in action:

**1. Clone & Build**
```bash
go mod tidy
go build -o rate-limiter ./cmd/limiter
```

**2. Spin up the 5-Node Demo Cluster**
```bash
docker compose up -d --build
docker compose logs -f
```

**3. Run the Performance Benchmarks**
```bash
go test -v -bench=. ./...
```

**4. Run Fault Injection (Partition Simulation)**
```bash
sudo ./scripts/jepsen_test.sh
```

---

## 🔒 Security Posture

For production deployments, node-to-node gossip automatically supports **Mutual TLS (mTLS)**. By loading certificates securely into `internal/netutil/mtls.go`, incoming malicious Gossip packets (e.g., trying to artificially drain tokens) are aggressively rejected at the transport layer.
