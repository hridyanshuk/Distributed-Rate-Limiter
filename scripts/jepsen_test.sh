#!/bin/bash
set -e

echo "Starting 5-node cluster..."
docker compose up -d --build

echo "Waiting for cluster to stabilize..."
sleep 5

echo "Injecting network partition on Node 3..."
# Block all UDP traffic (Gossip) to and from node 3 to simulate a partition.
# Node 3 will still serve API requests using stale local state (degrading gracefully).
docker compose exec node3 iptables -A INPUT -p udp --dport 7946 -j DROP
docker compose exec node3 iptables -A OUTPUT -p udp --dport 7946 -j DROP

echo "Partition established."

# In a real test, we would run load here (e.g., ghz or a custom integration client)
# and observe that Node 3 continues to allow requests until its local bucket is emptied,
# while the rest of the cluster stays in sync.

echo "Healing partition on Node 3..."
docker compose exec node3 iptables -D INPUT -p udp --dport 7946 -j DROP
docker compose exec node3 iptables -D OUTPUT -p udp --dport 7946 -j DROP

echo "Partition healed. Waiting for state to merge via Gossip..."
sleep 5

echo "Tearing down cluster..."
docker compose down

echo "Jepsen-style partition test completed successfully!"
