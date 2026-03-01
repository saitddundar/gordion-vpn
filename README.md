# Gordion VPN

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Status](https://img.shields.io/badge/status-experimental-orange)](#development-status)

A decentralized, peer-to-peer VPN built with a microservice control plane and a WireGuard-based data plane.

## Overview

Gordion VPN turns participating nodes into both clients and relay peers. It uses a modern microservices control plane (Identity, Discovery, Config) to coordinate a fully encrypted WireGuard mesh, aiming for secure, scalable, and geographically distributed VPN connectivity.

## Key Features

- **Decentralized topology**: Each node can act as both a client and a relay, reducing reliance on a single choke point.
- **WireGuard data plane**: Curve25519-based key exchange with a minimal, modern VPN protocol.
- **Microservice control plane**: Identity, discovery, and configuration services are isolated and independently deployable.
- **Strong authentication**: JWT-based node identities with inter-service token validation.
- **Observability by default**: Prometheus metrics, structured logging, and trace ID propagation across services.
- **Resilience focus**: Health checks, exponential backoff, graceful shutdown, and rate limiting integrated into the core services.

## Use Cases

- **Self-hosted, privacy-focused VPN** where you control both the control plane and the data plane.
- **Mesh-style connectivity** between multiple regions, offices, or homelabs without central VPN appliances.
- **Experimentation platform** for distributed systems concepts: service discovery, tracing, rate limiting, and resilience patterns.
- **Educational reference** for a production-style Go microservices stack with gRPC, WireGuard, and observability tooling.

## Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                       CONTROL PLANE                          │
│                                                              │
│  ┌──────────────┐  ┌────────────────┐  ┌────────────────┐    │
│  │   Identity   │  │   Discovery    │  │     Config     │    │
│  │   Service    │  │    Service     │  │    Service     │    │
│  │  PostgreSQL  │  │     Redis      │  │     Redis      │    │
│  │   JWT Auth   │  │ Peer Registry  │  │ IP Allocator   │    │
│  └──────────────┘  └────────────────┘  └────────────────┘    │
│         ↑                  ↑                   ↑             │
│  ┌──────────────────────────────────────────────────────┐    │
│  │            Prometheus + Grafana                      │    │
│  │         Distributed Tracing (trace_id)               │    │
│  │         Rate Limiting & Health Checks                │    │
│  └──────────────────────────────────────────────────────┘    │
└──────────────────────────────────────────────────────────────┘
           ↑                 ↑                   ↑
           │          gRPC + TLS                 │
┌──────────────────────────────────────────────────────────────┐
│                        DATA PLANE                            │
│                                                              │
│  ┌──────────┐                             ┌──────────┐       │
│  │  Agent   │ ◄══ WireGuard Tunnel ══►    │  Agent   │       │
│  │ (Node A) │      Curve25519 Keys        │ (Node B) │       │
│  └──────────┘                             └──────────┘       │
└──────────────────────────────────────────────────────────────┘
```

### Control Plane (Microservices)

| Service | Port | Status | Description |
|---------|------|--------|-------------|
| **Identity Service** | 8001 | Complete | Node authentication, JWT token management |
| **Discovery Service** | 8002 | Complete | Peer discovery, registration, heartbeat |
| **Config Service** | 8003 | Complete | Network configuration, IP allocation |

### Data Plane

| Component | Status | Description |
|-----------|--------|-------------|
| **Agent** | Complete | VPN client with WireGuard tunnel management |

### Observability Stack

| Tool | Port | Description |
|------|------|-------------|
| **Prometheus** | 9093 | Scrapes metrics from all services |
| **Grafana** | 3000 | Dashboards (admin / admin) |
| **Health Checks** | gRPC | Standard `grpc.health.v1` protocol on every service |
| **Rate Limiting** | - | Per-IP sliding window, 100 req/min default |
| **Distributed Tracing** | - | `x-trace-id` propagation via gRPC metadata |

## Project Structure

```
gordion-vpn/
├── services/
│   ├── identity/              # Identity service (PostgreSQL)
│   │   ├── cmd/server/        # Entry point
│   │   ├── internal/          # Config, storage, service, gRPC handler
│   │   ├── migrations/        # Database migrations
│   │   ├── Dockerfile
│   │   └── test/              # Integration tests
│   ├── discovery/             # Discovery service (Redis)
│   │   ├── cmd/server/        # Entry point
│   │   ├── internal/          # Config, registry, matcher, gRPC handler
│   │   ├── Dockerfile
│   │   └── test/              # Integration tests
│   ├── config/                # Config service (Redis)
│   │   ├── cmd/server/        # Entry point
│   │   ├── internal/          # Config, allocator, gRPC handler
│   │   ├── Dockerfile
│   │   └── test/              # Integration tests
│   └── agent/                 # VPN agent (WireGuard)
│       ├── cmd/agent/         # Entry point
│       ├── internal/
│       │   ├── agent/         # Lifecycle orchestration
│       │   ├── client/        # gRPC client for all services
│       │   ├── config/        # Agent configuration
│       │   ├── p2p/           # libp2p host, hole punching, WG bridge
│       │   └── wireguard/     # Tunnel management, key generation
│       ├── Dockerfile
│       └── test/              # End-to-end integration tests
├── pkg/
│   ├── auth/                  # Inter-service authentication client
│   ├── circuitbreaker/        # Circuit breaker for gRPC client resilience
│   ├── config/                # Configuration management
│   ├── grpcutil/              # gRPC error utilities
│   ├── healthcheck/           # gRPC health check utilities
│   ├── logger/                # Structured logging (zerolog)
│   ├── metrics/               # Prometheus metrics
│   ├── middleware/            # Logging interceptor (request_id)
│   ├── ratelimit/             # Per-IP sliding window rate limiter
│   ├── tlsutil/               # TLS credential helpers
│   ├── tracing/               # Distributed tracing (trace_id propagation)
│   └── proto/                 # Generated protobuf code
├── api/proto/                 # Protocol Buffer definitions
│   ├── identity/v1/
│   ├── discovery/v1/
│   └── config/v1/
├── deployments/               # Docker Compose, Prometheus config
├── configs/                   # Service configuration files
├── docs/                      # Documentation (architecture notes, design decisions)
├── scripts/                   # Proto generation, cert generation
└── Makefile                   # Build automation
```

## P2P Data Plane

Gordion VPN's data plane is designed to be **truly serverless** — after initial bootstrapping via the control plane, agents communicate directly with each other using **libp2p** as the transport layer.

### Phase 1: Bootstrap & Peer Discovery

```
Agent A                    Control Plane               Agent B
   │                            │                          │
   ├──RegisterNode(publicKey)──►│                          │
   │◄──(nodeID, token)──────────┤                          │
   ├──GetConfig(token)─────────►│                          │
   │◄──(vpn_ip, cidr)───────────┤                          │
   ├──RegisterPeer(ip, peerID)─►│                          │
   │                            │◄─RegisterPeer(ip,peerID)─┤
   ├──ListPeers()──────────────►│                          │
   │◄──[{ip, peerID, p2pAddrs}]─┤                          │
   │                            │                          │
   └─────── libp2p Ping (RTT check) ────────────────────►  │
```

Each agent registers its **libp2p PeerID** and multiaddresses alongside its WireGuard public key. This enables NAT traversal via **AutoNAT** and **Hole Punching** before WireGuard tunnel setup.

### Phase 2: WireGuard ↔ libp2p Bridge

The core challenge with P2P VPNs is NAT — two agents behind residential routers cannot directly exchange WireGuard UDP packets. The solution is to **tunnel WireGuard traffic over the libp2p stream**, which has already punched through NAT.

```
┌─────────────────────────────────────┐
│              Agent A                │
│                                     │
│  WireGuard TUN ──► Local UDP Sock   │
│   (10.8.0.2)       (127.0.0.1:X)    │
│         ▲               │           │
│         │      Bridge   ▼           │
│         └────────── libp2p Stream  ─┼──► (over internet, NAT punched)
└─────────────────────────────────────┘
                                          ┌─────────────────────────────────────┐
                                          │              Agent B                │
                                          │                                     │
                                          │  libp2p Stream ──► Local UDP Sock   │
                                          │                     (127.0.0.1:Y)   │
                                          │                          │          │
                                          │                          ▼          │
                                          │              WireGuard TUN          │
                                          │               (10.8.0.3)            │
                                          └─────────────────────────────────────┘
```

**How it works:**

1. **Custom Protocol** — A `/gordion/wg/1.0.0` libp2p protocol is registered on each agent. When two agents discover each other, one opens a bidirectional stream using this protocol.
2. **Per-Peer Proxy Ports** — Each peer gets a dedicated local UDP port (e.g., peer B → `:51920`, peer C → `:51921`). WireGuard's endpoint is set to this loopback address, ensuring packets for different peers never mix.
3. **Length-Prefixed Framing** — UDP datagrams are encapsulated with a 2-byte big-endian length header before being written to the libp2p stream. This prevents packet boundary loss, which is critical because libp2p streams are TCP-like (byte-oriented), while WireGuard expects discrete UDP datagrams.
4. **Outgoing Bridge** — A goroutine reads WireGuard packets from the peer's proxy port → prepends the length header → writes them into the specific libp2p stream.
5. **Incoming Bridge** — A goroutine reads the length header from the libp2p stream → reads exactly that many bytes → writes the reconstructed UDP datagram to WireGuard's listen port.
6. **Stream Race Prevention** — Only the peer with the lexicographically larger PeerID initiates the stream. The other side waits for the incoming connection. This deterministic rule eliminates duplicate streams and race conditions.
7. **Bridge Lifecycle** — If a libp2p stream drops (network change, peer disconnect), the bridge goroutines exit cleanly. On the next `peerSyncLoop` cycle, the agent rediscovers the peer and re-establishes the bridge automatically.

**Performance note:** The libp2p layer adds minimal overhead — it only transports already-encrypted WireGuard UDP datagrams. There is no double encryption; WireGuard's Curve25519 + ChaCha20-Poly1305 handles all payload security.

### Peer Selection Strategy

When multiple peers are available, the agent prioritizes by:

| Priority | Criterion | Reason |
|----------|-----------|--------|
| 1 | **libp2p Ping RTT** | Lowest round-trip time = best performance |
| 2 | **Reported Bandwidth** | Prefer high-bandwidth relay nodes |
| 3 | **Region match** | Reduce geographic latency |
| 4 | **Last Heartbeat** | Prefer recently active nodes |



## Getting Started

### Prerequisites

- Go 1.25+
- Docker and Docker Compose
- Protocol Buffers compiler (`protoc`)
- Make (optional, for build automation)

### Quick Start

```bash
# Clone the repository
git clone https://github.com/saitddundar/gordion-vpn.git
cd gordion-vpn

# Start infrastructure (PostgreSQL, Redis, Prometheus, Grafana)
docker-compose -f deployments/docker-compose.dev.yml up -d postgres redis

# Run database migrations
docker exec -i gordion-postgres psql -U gordion -d gordion \
  < services/identity/migrations/0001_initial.sql
docker exec -i gordion-postgres psql -U gordion -d gordion \
  < services/identity/migrations/0002_add_peer_id.sql

# Start services (each in a separate terminal)
cd services/identity  && go run ./cmd/server
cd services/discovery && METRICS_PORT=9094 go run ./cmd/server
cd services/config    && METRICS_PORT=9095 go run ./cmd/server

# Start two agent instances to test P2P (optional)
cd services/agent && P2P_PORT=4001 WIREGUARD_PORT=51820 go run ./cmd/agent
cd services/agent && P2P_PORT=4002 WIREGUARD_PORT=51821 go run ./cmd/agent
```

### Local Development Tips

- Run `make proto` after editing any file under `api/proto` to regenerate gRPC stubs in `pkg/proto`.
- Use `make tidy-all` to keep every module's `go.mod` in sync when you add a dependency.
- Export `GORDION_ENV=dev` (or pass `--env dev`) so each service picks up the matching file in `configs/`.
- `make docker-logs` (or `docker compose logs -f` inside `deployments/`) is handy for tailing Postgres/Redis while testing the agent.

### Config Hot-Reload

The **Config Service** supports zero-downtime configuration updates via `SIGHUP`:

```bash
# Modify configs/config.dev.yaml
# Then send SIGHUP to the config service process
kill -SIGHUP <pid>
```

The service will re-read the configuration and increment the **Config Version**, which agents will automatically detect on their next refresh.

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `JWT_SECRET` | (from yaml) | JWT signing secret |
| `DATABASE_URL` | (from yaml) | PostgreSQL connection string |
| `IDENTITY_ADDR` | `localhost:8001` | Identity service address |
| `DISCOVERY_ADDR` | `localhost:8002` | Discovery service address |
| `CONFIG_ADDR` | `localhost:8003` | Config service address |
| `TLS_CERT` | - | TLS certificate path (optional) |
| `TLS_KEY` | - | TLS private key path (optional) |
| `LOG_LEVEL` | `debug` | Log level (debug, info, warn, error) |

## Security

### Authentication Flow

```
Agent → Identity Service: RegisterNode(public_key)
                       ← token + node_id

Agent → Config Service: GetConfig(token)
Config → Identity: ValidateToken(token) ← inter-service auth
                ← network config

Agent → Discovery: RegisterPeer(token, ip, port)
Discovery → Identity: ValidateToken(token) ← inter-service auth
                   ← success
```

### Security Layers

| Layer | Description |
|---|---|
| Node Authentication | JWT tokens via Identity Service |
| Inter-Service Auth | Token validation between services |
| Transport Security | Optional TLS for gRPC (cert generation via `scripts/gen-certs.ps1`) |
| Tunnel Encryption | WireGuard with Curve25519 key exchange |
| Abuse Prevention | Per-IP Rate Limiting (100 req/min default) |
| Secret Management | Environment variable overrides, `.env` support |

## Agent Lifecycle

```
Start:
  1. Start libp2p P2P host (unique PeerID, Hole Punching enabled)
  2. Generate WireGuard keypair (Curve25519)
  3. Register with Identity Service (with **Exponential Backoff** + PeerID) → get token
  4. Fetch network config (supports **Config Versioning**)
  5. Request VPN IP address
  6. Announce to Discovery Service (IP, Port, PeerID, P2P multiaddrs)
  7. Discover other peers → attempt **libp2p Handshake (Ping)** per peer
  8. Fetch peer WireGuard public keys from Identity Service
  9. Configure WireGuard tunnel (real tunnel or dry-run)
 10. Start background loops:
     - **Heartbeat Loop**: Keeps peer status alive in Discovery
     - **Token Refresh Loop**: Automatically re-registers at 80% of token life

Shutdown (Ctrl+C):
  1. **Graceful Shutdown**: Wait for in-flight requests (10s timeout)
  2. Stop heartbeat & refresh loops
  3. Release VPN IP from Config Service
  4. Tear down WireGuard tunnel
  5. Close libp2p host
  6. Close all gRPC connections
```

## Observability

### Metrics Endpoints

All services expose Prometheus metrics on dedicated HTTP ports.

| Service | Metrics Port | Override |
|---------|-------------|----------|
| Identity | `9090` | `METRICS_PORT` env |
| Discovery | `9094` | `METRICS_PORT` env |
| Config | `9095` | `METRICS_PORT` env |
| Prometheus | `9093` | docker-compose |
| Grafana | `3000` | docker-compose (admin/admin) |

| Metric | Description |
|--------|-------------|
| `gordion_grpc_requests_total` | Total gRPC requests by service, method, status |
| `gordion_grpc_request_duration_seconds` | Request latency histogram |
| `gordion_active_connections` | Active connections per service |
| `gordion_db_queries_total` | Database query count |
| `gordion_db_query_duration_seconds` | Database query latency |

### Distributed Tracing

Trace IDs propagate via gRPC metadata (`x-trace-id`). A single request is traceable across all services:

```
Agent [trace: a3f29b01] → Identity [trace: a3f29b01] → Config [trace: a3f29b01]
```

### Structured Logging

Each request gets a unique `request_id`. Combined with `trace_id`:

```
[a3f29b01] [identity] --> /identity.v1.IdentityService/RegisterNode
[a3f29b01] [identity] <-- /identity.v1.IdentityService/RegisterNode | OK
```

## Makefile Commands

```bash
make help              # Show all available commands

# Build
make build-all         # Build all services
make build-identity    # Build identity service
make build-discovery   # Build discovery service
make build-config      # Build config service

# Test
make test-all          # Run all integration tests
make test-identity     # Run identity tests
make test-discovery    # Run discovery tests
make test-config       # Run config tests

# Docker
make docker-up         # Start infrastructure
make docker-down       # Stop infrastructure
make docker-restart    # Restart infrastructure

# Utilities
make tidy-all          # Run go mod tidy on all modules
make proto             # Generate protobuf code
make clean             # Remove build artifacts
```

## Testing

```bash
# Run all tests
make test-all

# Or individually
cd services/identity  && go test -v -count=1 ./test/...
cd services/discovery && go test -v -count=1 ./test/...
cd services/config    && go test -v -count=1 ./test/...
cd services/agent     && go test -v -count=1 ./test/...
```

## API Reference

### Identity Service (port 8001)

| Method | Request | Response |
|--------|---------|----------|
| `RegisterNode` | `public_key`, `version`, `peer_id` | `node_id`, `token`, `expires_at` |
| `ValidateToken` | `token` | `valid`, `node_id` |
| `GetPublicKey` | `node_id` | `public_key` |

### Discovery Service (port 8002)

| Method | Request | Response |
|--------|---------|----------|
| `RegisterPeer` | `token`, `ip_address`, `port`, `peer_id`, `p2p_addrs[]` | `success`, `message` |
| `ListPeers` | `region`, `limit` | `peers[]` (includes `peer_id`, `p2p_addrs`) |
| `Heartbeat` | `token`, `bandwidth` | `success`, `ttl` |

### Config Service (port 8003)

| Method | Request | Response |
|--------|---------|----------|
| `GetConfig` | `token`, `config_version` | `network_cidr`, `mtu`, `dns_servers`, `config_version`, `up_to_date` |
| `RequestIP` | `token`, `node_id` | `ip_address`, `subnet_mask`, `gateway` |
| `ReleaseIP` | `token`, `node_id`, `ip_address` | `success`, `message` |

## Tech Stack

| Category | Technology |
|----------|-----------|
| Language | Go 1.25 |
| RPC | gRPC, Protocol Buffers |
| Database | PostgreSQL 15 |
| Cache | Redis 7 |
| Authentication | JWT (HMAC-SHA256) |
| Encryption | WireGuard, Curve25519 |
| Logging | zerolog (structured) |
| Metrics | Prometheus, Grafana |
| Tracing | Custom trace_id propagation via gRPC metadata |
| Transport | Optional TLS for gRPC |
| Build | Make, Multi-stage Docker |
| Containers | Docker, Docker Compose |

## Development Status

### Sprint 1: Foundation ✅
- Monorepo setup, proto definitions, shared packages

### Sprint 2: Identity Service ✅
- PostgreSQL storage, JWT authentication, gRPC API, integration tests, Prometheus metrics

### Sprint 3: Discovery Service ✅
- Redis registry, peer matching, heartbeat mechanism, gRPC API, integration tests, Prometheus metrics

### Sprint 4: Config Service ✅
- IP allocation (DHCP-like), network configuration, gRPC API, integration tests, Prometheus metrics

### Sprint 5: Agent ✅
- WireGuard tunnel management, Curve25519 key generation, full lifecycle orchestration, end-to-end tests

### Sprint 6: Security & Observability ✅
- Inter-service authentication, optional TLS, secret management, structured logging, distributed tracing

### Sprint 7: Resilience & Polish ✅
- gRPC Health Check, Per-IP Rate Limiting, Exponential backoff, Token refresh
- SIGHUP hot-reload, Config versioning, Graceful shutdown

### Sprint 8: P2P Foundation ✅
- libp2p host per agent (unique PeerID, Noise encryption)
- NAT Traversal: AutoNAT + Hole Punching enabled
- Peer discovery extended: peer_id & p2p_addrs stored in Identity DB and Discovery Redis
- P2P Handshake (Ping) verified between two local agents (RTT ~0ms loopback)
- CI/CD pipeline: GitHub Actions (build, integration tests, lint, govulncheck)

### Sprint 9: WireGuard ↔ libp2p Bridge ✅
- `/gordion/wg/1.0.0` custom libp2p protocol
- Per-peer UDP proxy ports (no broadcast, each peer has dedicated relay)
- Length-prefixed framing for UDP-over-stream
- PeerID-based stream initiator to prevent race conditions
- Full integration with peerSyncLoop for automatic bridge on new peer discovery
- WireGuard ListenPort + CIDR address format fixes

## Challenges & Solutions

During the development of Gordion VPN, we encountered several architectural and network-level challenges. Documenting these ensures the project evolves as an enterprise-grade solution rather than just a hobby project.

### 1. The NAT Traversal Problem (Hole Punching)
**Challenge:** 
When two agents attempt to establish a P2P WireGuard tunnel, they usually reside behind strictly configured NATs (Network Address Translation) and residential routers. Direct communication drops because their `192.168.x.x` IPs are non-routable over the internet, and modem firewalls block incoming ping requests.

**Solution:**
Instead of relaying all traffic through a slow central proxy, we integrated the **libp2p** networking stack into the Agent:
- **AutoNAT & STUN:** The agent runs an AutoNAT service on boot to discover its *true* public IP and port from the perspective of the outside world.
- **Hole Punching:** We utilized `libp2p.EnableHolePunching()` to coordinate simultaneous connection attempts from both peers. By doing a quick P2P Handshake (Ping) *before* configuring the WireGuard tunnel, we punch a hole through the NAT, allowing the WireGuard UDP packets to flow directly (true P2P).

### 2. Centralized vs Decentralized State
**Challenge:** 
If the central `Discovery Service` dies, newly joined agents wouldn't know who to connect to, creating a single point of failure.

**Solution (Hybrid Architecture):** 
Gordion VPN uses a **Hybrid Control Plane**. Centralized microservices (Identity, Config, Discovery) handle global authentication, policy, and act as "Bootstrap Nodes". Once the initial peer list is acquired, agents communicate directly via libp2p.

### 3. Per-Peer Packet Routing
**Challenge:**
The initial bridge design broadcast every WireGuard packet to all connected peers. With 10 peers, each packet was sent 10 times — 9 copies wasted.

**Solution:**
Each peer is assigned a dedicated local UDP proxy port. WireGuard’s peer endpoint points to the specific proxy, so packets are routed 1:1 to the correct libp2p stream. No broadcast, no wasted bandwidth.

## License

MIT License - see [LICENSE](LICENSE) file for details.
