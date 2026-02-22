# API Presentation Layer

The Sub-Millisecond Distributed Rate Limiter is designed to be queried by an API Gateway (like Envoy, NGINX, or custom Go/Rust gateways). To accommodate various environments, the system exposes two interfaces.

## 1. gRPC Interface (Standard)

For maximum compatibility, strong typing, and ease of developer integration, the system exposes a standard unary gRPC endpoint.

### Protobuf Schema

Located in `api/v1/ratelimit.proto`:

```protobuf
syntax = "proto3";
package ratelimit.v1;

service RateLimiter {
  // AllowRequest checks if a given client ID is allowed to make a request.
  rpc AllowRequest(AllowRequestArgs) returns (AllowRequestResponse) {}
}

message AllowRequestArgs {
  string client_id = 1;
}

message AllowRequestResponse {
  bool allowed = 1;
}
```

### Usage
Clients connect to `:50051` (default) and fire `AllowRequest`. 

Behind the scenes:
1. The gRPC handler extracts `client_id` and hits the local `Store`.
2. The `TokenBucket` uses lock-free atomics to consume 1 token.
3. If successful, the gateway responds `Allowed: true`, and the cluster background-syncs the consumed token via SWIM.

## 2. Raw UDP Interface (Ultra-Low Latency)

For FAANG-scale applications where even the gRPC/Protobuf overhead (marshal/unmarshal operations on the heap) is considered too slow, the system provides an alternative raw UDP server.

### Protocol
The UDP interface is completely stateless and connectionless, maximizing throughput.

**Request Format:**
- Send the raw string of the `client_id` as bytes inside a UDP datagram to port `:6000` (default).

**Response Format:**
- The server responds strictly with `1 byte` to the sender's IP/Port:
  - `0x01` -> **Allowed**
  - `0x00` -> **Denied** (Rate Limited)

### Usage Example (Go Client)

```go
conn, _ := net.Dial("udp", "ratelimiter-node:6000")
defer conn.Close()

// Query for client token
conn.Write([]byte("api_key_12345"))

// Get 1-byte response
resp := make([]byte, 1)
conn.Read(resp)

if resp[0] == 1 {
    // Proceed with HTTP Request...
} else {
    // Return HTTP 429 Too Many Requests...
}
```

### Why two interfaces?
- **gRPC** offers connection pooling, retries, Load Balancing integrations (e.g., Envoy `ext_authz`), and structured responses (e.g., allowing returning exactly *how many* tokens remain or reset headers).
- **UDP** bypasses TCP handshake overhead, TLS overhead, and Protobuf instantiation for internal hyper-optimized services that just need a binary "Go/No-Go" gate.
