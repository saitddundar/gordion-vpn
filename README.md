# Gordion VPN

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Status](https://img.shields.io/badge/status-active--development-orange)](#development-status)
[![CI](https://github.com/saitddundar/gordion-vpn/actions/workflows/ci.yml/badge.svg)](https://github.com/saitddundar/gordion-vpn/actions)
[![Views](https://hits.seeyoufarm.com/api/count/incr/badge.svg?url=https%3A%2F%2Fgithub.com%2Fsaitddundar%2Fgordion-vpn&count_bg=%2300ADD8&title_bg=%23555555&icon=github.svg&icon_color=%23FFFFFF&title=Views&edge_flat=false)](https://hits.seeyoufarm.com)

A self-hosted, peer-to-peer mesh VPN built with Go. Connects devices securely across NAT boundaries using WireGuard tunnels transported over libp2p — with optional exit node support for internet privacy.

## Table of Contents

- [Overview](#overview)
- [Key Features](#key-features)
- [Architecture](#architecture)
- [Project Structure](#project-structure)
- [P2P Data Plane](#p2p-data-plane)
- [CLI — `gordion`](#cli--gordion)
- [Getting Started](#getting-started)
- [Agent Configuration](#agent-configuration)
- [Security](#security)
- [Agent Lifecycle](#agent-lifecycle)
- [Observability](#observability)
- [Makefile Commands](#makefile-commands)
- [Testing](#testing)
- [API Reference](#api-reference)
- [Challenges & Solutions](#challenges--solutions)
- [License](#license)

## Overview

Gordion VPN turns participating nodes into both clients and relay peers. It uses a modern microservices control plane (Identity, Discovery, Config) to coordinate a fully encrypted WireGuard mesh, with NAT traversal handled by libp2p hole punching.

**Two modes:**

| Mode | What it does |
|------|-------------|
| **Mesh VPN** | Connect your devices into a private encrypted network across NAT/firewalls (like Tailscale, self-hosted) |
| **Exit Node** | Route all internet traffic through a designated VPS — hiding your IP from sites and your ISP (like NordVPN, self-hosted) |

## Key Features 

- **Automatic peer discovery** — agents find and connect to each other without manual key exchange
- **NAT traversal** — connects devices behind home routers via libp2p hole punching (no relay needed)
- **WireGuard data plane** — Curve25519 key exchange, ChaCha20-Poly1305 payload encryption
- **Exit node support** — route all internet traffic through a VPS for IP masking and geo-bypass
- **DNS leak protection** — DNS queries route through the VPN tunnel when exit node is active
- **TLS for control plane** — gRPC connections between agent and services are optionally TLS-encrypted
- **Authenticated peer listing** — peer enumeration requires a valid JWT token (no anonymous scanning)
- **Observability** — Prometheus metrics, structured logging (zerolog), distributed trace IDs
- **Resilience** — circuit breakers, exponential backoff, rate limiting, graceful shutdown, SIGHUP config hot-reload

## Architecture
```text
┌───────────────────────────────────────────────────────────────────────────────┐
│                                                                               │
│                             CONTROL PLANE (Microservices)                     │
│                                                                               │
│  ┌─────────────────────┐   ┌─────────────────────┐   ┌─────────────────────┐  │
│  │   Identity Service  │   │  Discovery Service  │   │   Config Service    │  │
│  │---------------------│   │---------------------│   │---------------------│  │
│  │ • Node PKI          │   │ • Peer Registry     │   │ • IP Allocator      │  │
│  │ • JWT Generation    │   │ • Heartbeat         │   │ • Network CIDR      │  │
│  │                     │   │                     │   │                     │  │
│  │   [( PostgreSQL )]  │   │     [( Redis )]     │   │     [( Redis )]     │  │
│  └──────────┬──────────┘   └──────────┬──────────┘   └──────────┬──────────┘  │
│             │                         │                         │             │
│  ┌──────────▼─────────────────────────▼─────────────────────────▼──────────┐  │
│  │                      Security, Middleware & Resilience                  │  │
│  │    [ JWT Validation ]      [ Circuit Breakers ]     [ Rate Limiting ]   │  │
│  └────────────────────────────────────┬────────────────────────────────────┘  │
│                                       │                                       │
└───────────────────────────────────────┼───────────────────────────────────────┘
                                        │
                       gRPC + TLS (Bootstrap & Heartbeat)
                                        │
┌───────────────────────────────────────▼───────────────────────────────────────┐
│                                                                               │
│  ┌──────────────────┐       libp2p Stream Tunnel      ┌──────────────────┐    │
│  │    Agent A       │◄═══════════════════════════════►│    Agent B(VPS)  │    │
│  │  (Client Node)   │       WireGuard Encryption      │   [Exit Node]    │    │
│  │  VPN IP: .../32  │      AutoNAT & Hole Punched     │   VPN IP: .../32 │    │
│  └──────────────────┘                                 └─────────┬────────┘    │
│                                                                 │             │
│                          DATA PLANE (Zero-Trust P2P Mesh)       │             │
└─────────────────────────────────────────────────────────────────┼─────────────┘
                                                                  │
                                                        iptables MASQUERADE & 
                                                           DNS Resolution     
                                                                  │
                                                                  ▼
                                                       (( Public Internet ))

  OBSERVABILITY STACK:
  • Prometheus + Grafana (Metrics exported from all services)
  • Distributed Tracing (x-trace-id context across all layers)
  • Structured JSON Logging (Zerolog integration)
```

### Control Plane Services

| Service | Port | Storage | Description |
|---------|------|---------|-------------|
| **Identity Service** | 8001 | PostgreSQL | Node registration, JWT token issuance & validation, public key storage |
| **Discovery Service** | 8002 | Redis | Peer registry, exit node announcements, heartbeat, authenticated ListPeers |
| **Config Service** | 8003 | Redis | Network CIDR, IP allocation (DHCP-like), DNS config, SIGHUP hot-reload |

### Data Plane

| Component | Description |
|-----------|-------------|
| **Agent** | VPN client: WireGuard tunnel + libp2p P2P host + gateway (exit node NAT) |

### Observability Stack

| Tool | Port | Description |
|------|------|-------------|
| **Prometheus** | 9093 | Scrapes metrics from all services |
| **Grafana** | 3000 | Dashboards (admin / admin) |
| **Health Checks** | gRPC | Standard `grpc.health.v1` on every service |
| **Rate Limiting** | — | Per-IP sliding window, 100 req/min default |
| **Distributed Tracing** | — | `x-trace-id` propagation via gRPC metadata |

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
│   └── agent/                 # VPN agent (WireGuard + libp2p)
│       ├── cmd/agent/         # Entry point
│       ├── internal/
│       │   ├── agent/         # Lifecycle orchestration
│       │   ├── client/        # gRPC client (TLS-aware, circuit breaker)
│       │   ├── config/        # Agent configuration (YAML + env override)
│       │   ├── gateway/       # Exit node: IP forwarding + NAT (Linux/Windows/macOS)
│       │   ├── p2p/           # libp2p host, hole punching, WireGuard bridge
│       │   └── wireguard/     # Tunnel management, key generation (0600 permissions)
│       ├── Dockerfile
│       └── test/              # End-to-end integration tests
├── pkg/
│   ├── auth/                  # Inter-service JWT validation client
│   ├── circuitbreaker/        # Circuit breaker for gRPC client resilience
│   ├── config/                # Configuration management
│   ├── grpcutil/              # gRPC error utilities
│   ├── healthcheck/           # gRPC health check utilities
│   ├── logger/                # Structured logging (zerolog)
│   ├── metrics/               # Prometheus metrics
│   ├── middleware/            # Logging interceptor (request_id)
│   ├── ratelimit/             # Per-IP sliding window rate limiter
│   ├── tlsutil/               # TLS credential helpers (server + client)
│   ├── tracing/               # Distributed tracing (trace_id propagation)
│   └── proto/                 # Generated protobuf code
├── api/proto/                 # Protocol Buffer definitions
│   ├── identity/v1/
│   ├── discovery/v1/
│   └── config/v1/
├── deployments/               # Docker Compose, Prometheus config
├── configs/                   # Service configuration files
├── scripts/                   # Proto generation, cert generation
└── Makefile                   # Build automation
```

## P2P Data Plane

Gordion VPN's data plane is designed to be **truly P2P** — after initial bootstrapping via the control plane, agents communicate directly using **libp2p** as the transport layer.

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
   ├──ListPeers(token)─────────►│                          │
   │◄──[{ip, peerID, p2pAddrs,isExitNode}]─────────────────┤
   │                            │                          │
   └─────── libp2p Ping (RTT check) ────────────────────►  │
```

Each agent registers its **libp2p PeerID** and multiaddresses alongside its WireGuard public key. This enables NAT traversal via **AutoNAT** and **Hole Punching** before WireGuard tunnel setup.

### Phase 2: WireGuard ↔ libp2p Bridge

The core challenge with P2P VPNs is NAT — two agents behind home routers cannot directly exchange WireGuard UDP packets. The solution is to **tunnel WireGuard traffic over the libp2p stream**, which has already punched through NAT.

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
3. **Length-Prefixed Framing** — UDP datagrams are encapsulated with a **2-byte big-endian length header** before being written to the libp2p stream. This prevents packet boundary loss, which is critical because libp2p streams are TCP-like (byte-oriented), while WireGuard expects discrete UDP datagrams.
4. **Outgoing Bridge** — A goroutine reads WireGuard packets from the peer's proxy port → prepends the length header → writes them into the specific libp2p stream.
5. **Incoming Bridge** — A goroutine reads the length header from the libp2p stream → reads exactly that many bytes → writes the reconstructed UDP datagram to WireGuard's listen port.
6. **Stream Race Prevention** — Only the peer with the lexicographically larger PeerID initiates the stream. The other side waits for the incoming connection. This deterministic rule eliminates duplicate streams and race conditions.
7. **Bridge Lifecycle** — If a libp2p stream drops (network change, peer disconnect), the bridge goroutines exit cleanly. On the next `peerSyncLoop` cycle, the agent rediscovers the peer and re-establishes the bridge automatically.

> **Performance note:** The libp2p layer adds minimal overhead — it only transports already-encrypted WireGuard UDP datagrams. There is no double encryption; WireGuard's Curve25519 + ChaCha20-Poly1305 handles all payload security.

### Exit Node (Optional)

When `is_exit_node: true` is set on a peer (typically a VPS), it:
1. Announces itself to Discovery with `is_exit_node = true`
2. Enables kernel IP forwarding (`sysctl net.ipv4.ip_forward=1`)
3. Configures NAT masquerade (`iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE`)

When a client sets `use_exit_node: true`:
1. Finds an exit node from the peer list (specific by ID, or auto-selects first available)
2. Sets that peer's `AllowedIPs = 0.0.0.0/0, ::/0` — all internet traffic goes through it
3. Overrides WireGuard's `DNS` field (default: `1.1.1.1, 1.0.0.1`) — **DNS queries also go through the tunnel**, preventing DNS leaks to the local ISP resolver

```
Normal mode:   Device ──wg──► Peer    (private mesh, same internet IP)

Exit node:     Device ──wg──► VPS ──► Internet
                              ↑
               ISP only sees encrypted traffic to VPS
               Websites see VPS IP, not your real IP
               DNS queries go through the tunnel (no leaks)
```

### Peer Selection Strategy

When multiple peers are available, the agent prioritizes by:

| Priority | Criterion | Reason |
|----------|-----------|--------|
| 1 | **libp2p Ping RTT** | Lowest round-trip time = best performance |
| 2 | **Reported Bandwidth** | Prefer high-bandwidth relay nodes |
| 3 | **Region match** | Reduce geographic latency |
| 4 | **Last Heartbeat** | Prefer recently active nodes |

## CLI — `gordion`

The `gordion` CLI is the primary interface for managing the VPN on end nodes. It starts/stops the agent, shows status, lists peers, and runs diagnostics — all from a single binary.

### Installation

```bash
# Build and install to GOPATH/bin
make install-cli

# Or build locally
make build-cli          # produces cli/gordion.exe
```

### First-Time Setup

```bash
# Generate a WireGuard keypair and default config (run once per node)
gordion init --secret <your_network_secret>

# Edit the generated config to point at your control plane servers
nano configs/agent.yaml

# Start the VPN
gordion up
```

### Command Reference

| Command | Description |
|---------|-------------|
| `gordion up` | Start the VPN agent in the background |
| `gordion down` | Stop the agent (graceful, 10s timeout) |
| `gordion restart` | Restart the agent (down + up) |
| `gordion status` | Show connection status, VPN IP, uptime |
| `gordion peers` | List all peers in the network |
| `gordion exit-node list` | Show available exit nodes |
| `gordion exit-node set [id]` | Route internet traffic via an exit node |
| `gordion exit-node off` | Disable exit node routing |
| `gordion logs [-f] [-n N]` | View or stream agent logs |
| `gordion doctor` | Run connectivity diagnostics (7 checks) |
| `gordion init [--secret]` | Generate keypair + default config |
| `gordion version` | Print version, OS, arch, Go runtime |

**Global flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--config` | `-c` | Config file path (default: `configs/agent.dev.yaml`) |
| `--json` | `-j` | Output as JSON for scripting / dashboards |

### Examples

```bash
# Connect with a specific config
gordion up -c /etc/gordion/agent.yaml

# Check status as JSON (for scripts/dashboards)
gordion status -j
# → {"connected":true,"vpn_addr":"10.8.0.5/24","uptime_sec":3714,...}

# List peers as JSON
gordion peers -j

# Stream logs in real time
gordion logs -f

# Check all services are reachable
gordion doctor
# →  ✓  Agent process          Running (PID 12345)
# →  ✓  Identity Service       OK (localhost:8001)
# →  ✗  Discovery Service      unreachable (localhost:8002): ...

# Set an exit node
gordion exit-node set node-abc123

# Version with full build info
gordion version -j
# → {"version":"v0.2.0","os":"linux","arch":"amd64","go_version":"go1.22.0"}
```

### Security Note

`gordion init` stores the WireGuard **private key** in `~/.gordion/keys/private.key` with `0600` permissions. The key is **never** written into the config file — only the file path is referenced. Do not commit `~/.gordion/keys/` to version control.

---

## Getting Started

### Prerequisites

- Go 1.22+
- Docker and Docker Compose
- WireGuard (only for `dry_run: false` — real tunnel mode)
- Make (optional, for build automation)

### Quick Start

```bash
# Clone the repository
git clone https://github.com/saitddundar/gordion-vpn.git
cd gordion-vpn

# Start infrastructure (PostgreSQL, Redis, Prometheus, Grafana)
docker compose -f deployments/docker-compose.dev.yml up -d postgres redis

# Run database migrations
docker exec -i gordion-postgres psql -U gordion -d gordion \
  < services/identity/migrations/0001_initial.sql
docker exec -i gordion-postgres psql -U gordion -d gordion \
  < services/identity/migrations/0002_add_peer_id.sql

# Start services (each in a separate terminal)
cd services/identity  && go run ./cmd/server
cd services/discovery && METRICS_PORT=9094 go run ./cmd/server
cd services/config    && METRICS_PORT=9095 go run ./cmd/server

# Start two agent instances to test P2P (dry_run: true by default — no WireGuard needed)
cd services/agent && P2P_PORT=4001 WIREGUARD_PORT=51820 go run ./cmd/agent
cd services/agent && P2P_PORT=4002 WIREGUARD_PORT=51821 go run ./cmd/agent
```

### TLS Setup (Production)

```bash
# Generate self-signed dev certificates
.\scripts\gen-certs.ps1   # Windows
# bash scripts/gen-certs.sh  # Linux/macOS

# Then set in agent config:
# tls_ca_cert: "../../certs/ca-cert.pem"
```

Services detect `certs/server-cert.pem` at startup — TLS is enabled automatically if present; otherwise falls back to insecure mode (fine for local development).

### Local Development Tips

- Run `.\scripts\proto-gen.ps1` after editing any `.proto` file to regenerate gRPC stubs.
- Use `make tidy-all` to keep every module's `go.mod` in sync when adding a dependency.
- Export `GORDION_ENV=dev` so each service picks up the matching file in `configs/`.
- `docker compose logs -f` inside `deployments/` is handy for tailing Postgres/Redis while testing.

### Config Hot-Reload

The **Config Service** supports zero-downtime configuration updates via `SIGHUP`:

```bash
# Modify configs/config.dev.yaml, then:
kill -SIGHUP <pid>
```

The service re-reads the configuration and increments the **Config Version**, which agents automatically detect on their next refresh cycle.

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `JWT_SECRET` | (from yaml) | JWT signing secret |
| `NETWORK_SECRET` | (from yaml) | Registration required password |
| `DATABASE_URL` | (from yaml) | PostgreSQL connection string |
| `IDENTITY_ADDR` | `localhost:8001` | Identity service address |
| `DISCOVERY_ADDR` | `localhost:8002` | Discovery service address |
| `CONFIG_ADDR` | `localhost:8003` | Config service address |
| `TLS_CERT` | — | Server TLS certificate path |
| `TLS_KEY` | — | Server TLS private key path |
| `TLS_CA_CERT` | — | CA cert path for agent gRPC TLS |
| `IS_EXIT_NODE` | `false` | Run this agent as an exit node |
| `USE_EXIT_NODE` | `false` | Route internet traffic via exit node |
| `EXIT_NODE_ID` | `` | Specific exit node ID (empty = auto-select) |
| `EXIT_NODE_DNS` | `1.1.1.1, 1.0.0.1` | DNS used when exit node is active |
| `LOG_LEVEL` | `debug` | Log level (debug, info, warn, error) |

## Agent Configuration

```yaml
# configs/agent.dev.yaml
identity_addr: "localhost:8001"
discovery_addr: "localhost:8002"
config_addr:    "localhost:8003"

log_level: "debug"
heartbeat_interval: 25       # seconds
peer_sync_interval: 60       # seconds between peer discovery cycles

wireguard_port: 51820
p2p_port: 4001

dry_run: true   # true = log only, no real tunnel (useful without WireGuard installed)

# TLS: leave empty for insecure dev mode
# tls_ca_cert: "../../certs/ca-cert.pem"

# Exit Node Configuration:
# -- Mode 1 (normal peer, default):
is_exit_node: false
use_exit_node: false
exit_node_id: ""

# -- Mode 2 (this machine IS the exit node, e.g. a VPS):
# is_exit_node: true

# -- Mode 3 (use an exit node for internet privacy):
# use_exit_node: true
# exit_node_id: ""               # auto-select first available exit node
# exit_node_id: "node-abc123"    # or pin to a specific node
# exit_node_dns: "1.1.1.1, 1.0.0.1"  # DNS via tunnel (prevents DNS leaks)
```

## Security

### Authentication Flow

```
Agent → Identity Service: RegisterNode(public_key, network_secret)
                       ← token + node_id

Agent → Config Service: GetConfig(token)
Config → Identity: ValidateToken(token)   ← inter-service auth
                ← network config

Agent → Discovery: RegisterPeer(token, ip, port, is_exit_node)
Discovery → Identity: ValidateToken(token) ← inter-service auth
                   ← success

Agent → Discovery: ListPeers() [with token in gRPC metadata]
Discovery → Identity: ValidateToken(token) ← unauthenticated calls rejected
                   ← peer list
```

### Security Hardening Applied

| Concern | Fix |
|---------|-----|
| Unencrypted gRPC | Optional TLS — CA cert path enables encrypted transport for agent ↔ services |
| WireGuard private key world-readable | Written with `0600` permissions to `os.UserConfigDir()/gordion/` |
| Unauthenticated peer enumeration | `ListPeers` validates JWT from gRPC metadata |
| Subnet hijack via AllowedIPs | Each peer uses `/32` host route; only exit node gets `0.0.0.0/0` when explicitly selected |
| DNS leaks with exit node | WireGuard `DNS` field overridden to route queries through tunnel |
| Data races (token / config fields) | `sync.RWMutex` guards shared fields accessed across goroutines |
| Per-IP abuse | Rate limiting: 100 req/min sliding window on all services |

### Security Layers

| Layer | Description |
|-------|-------------|
| Node Authentication | JWT tokens via Identity Service |
| Inter-Service Auth | Token validation between services |
| Transport Security | Optional TLS for gRPC (cert generation via `scripts/gen-certs.ps1`) |
| Tunnel Encryption | WireGuard: Curve25519 key exchange + ChaCha20-Poly1305 |
| Peer Auth | Libp2p Noise protocol (authenticated key exchange on P2P layer) |
| Abuse Prevention | Per-IP sliding window rate limiting (100 req/min default) |
| Secret Management | Environment variable overrides, no secrets in config files |

## Agent Lifecycle

```
Start:
  1.  Start libp2p P2P host (unique PeerID, Noise encryption, Hole Punching)
  2.  Generate WireGuard keypair (Curve25519)
  3.  Register with Identity Service (Exponential Backoff + PeerID) → get token
  4.  Fetch network config (supports Config Versioning)
  5.  Request VPN IP address
  6.  Announce to Discovery Service (IP, Port, PeerID, P2P multiaddrs, is_exit_node)
  7.  If is_exit_node: enable IP forwarding + iptables NAT (gateway package)
  8.  Discover other peers → attempt libp2p Handshake (Ping) per peer
  9.  Fetch peer WireGuard public keys from Identity Service
 10.  Configure WireGuard tunnel (real tunnel or dry-run)
 11.  Start background loops:
       • Heartbeat Loop: keeps peer alive in Discovery
       • Token Refresh Loop: re-registers at 80% of token lifetime
       • Peer Sync Loop: discovers new peers, removes stale ones, updates WireGuard config

Shutdown (Ctrl+C / SIGTERM):
  1.  Graceful Shutdown: cancel context, wait for loops to exit (WaitGroup)
  2.  Release VPN IP from Config Service
  3.  Tear down WireGuard tunnel
  4.  If is_exit_node: remove iptables rules (gateway.Disable)
  5.  Close libp2p host + bridge streams
  6.  Close all gRPC connections
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
| `gordion_active_peers_total` | **[Domain]** Current number of online peers in VPN |
| `gordion_allocated_ips_total` | **[Domain]** Total number of IPs assigned to peers |
| `gordion_jwt_tokens_issued_total` | **[Domain]** Successful node registration count |
| `gordion_auth_failures_total` | **[Domain]** Failed auth attempts (wrong secret/token) |

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
make build-all         # Build all services + CLI
make build-identity    # Build identity service
make build-discovery   # Build discovery service
make build-config      # Build config service
make build-agent       # Build agent binary
make build-cli         # Build gordion CLI (gordion.exe)
make install-cli       # Install gordion to GOPATH/bin

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
make tidy-all          # Run go mod tidy on all modules (including CLI)
make proto             # Generate protobuf code
make vulncheck         # Scan all modules for CVEs
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
| `RegisterNode` | `public_key`, `version`, `peer_id`, `network_secret` | `node_id`, `token`, `expires_at` |
| `ValidateToken` | `token` | `valid`, `node_id` |
| `GetPublicKey` | `node_id` | `public_key` |

### Discovery Service (port 8002)

| Method | Auth | Request | Response |
|--------|------|---------|----------|
| `RegisterPeer` | token in body | `ip_address`, `port`, `peer_id`, `p2p_addrs`, `is_exit_node` | `success`, `message` |
| `ListPeers` | token in metadata | `region`, `limit` | `peers[]` (includes `peer_id`, `p2p_addrs`, `is_exit_node`) |
| `Heartbeat` | token in body | `bandwidth` | `success`, `ttl` |

### Config Service (port 8003)

| Method | Request | Response |
|--------|---------|----------|
| `GetConfig` | `token`, `config_version` | `network_cidr`, `mtu`, `dns_servers`, `config_version`, `up_to_date` |
| `RequestIP` | `token`, `node_id` | `ip_address`, `subnet_mask`, `gateway` |
| `ReleaseIP` | `token`, `node_id`, `ip_address` | `success`, `message` |

## Tech Stack

| Category | Technology |
|----------|-----------| 
| Language | Go 1.22+ |
| VPN Tunnel | WireGuard (Curve25519 + ChaCha20-Poly1305) |
| P2P Transport | libp2p (Noise, Yamux, AutoNAT, Hole Punching) |
| RPC | gRPC, Protocol Buffers |
| Database | PostgreSQL 15 |
| Cache | Redis 7 |
| Authentication | JWT (HMAC-SHA256) |
| Logging | zerolog (structured) |
| Metrics | Prometheus, Grafana |
| Tracing | Custom trace_id propagation via gRPC metadata |
| Transport Security | Optional TLS for gRPC |
| Resilience | Circuit breaker, exponential backoff, rate limiting |
| Build | Make, multi-stage Docker |
| Containers | Docker, Docker Compose |

## Development Status

### Completed 

| Sprint | Deliverables |
|--------|-------------|
| **Foundation** | Monorepo setup, proto definitions, shared packages |
| **Identity Service** | PostgreSQL storage, JWT authentication, gRPC API, integration tests, Prometheus metrics |
| **Discovery Service** | Redis registry, peer matching, heartbeat, authenticated ListPeers, integration tests |
| **Config Service** | IP allocation, network configuration, SIGHUP hot-reload, config versioning, integration tests |
| **Agent** | WireGuard tunnel management, Curve25519 key generation, full lifecycle, integration tests |
| **Observability** | Prometheus metrics, Grafana, distributed tracing, structured logging, health checks, rate limiting |
| **Resilience** | Circuit breaker, exponential backoff, token refresh loop, graceful shutdown |
| **P2P Foundation** | libp2p host per agent (PeerID, Noise), AutoNAT, Hole Punching, CI/CD pipeline |
| **WireGuard ↔ libp2p Bridge** | `/gordion/wg/1.0.0` protocol, UDP proxy ports, 2-byte length-prefix framing, race-free stream initiation |
| **Security Hardening** | Custom Network Secret auth, gRPC TLS support, WireGuard configs 0600, ListPeers auth, `/32` AllowedIPs |
| **Exit Node** | `is_exit_node` flag in Discovery, cross-os gateway package (iptables/netsh/pf), client-side exit node selection, DNS leak protection |

### Planned 

| Feature | Description |
|---------|-------------|
| **Web Dashboard** | Admin UI: peer management, invite links, exit node status |

## Challenges & Solutions

### 1. The NAT Traversal Problem (Hole Punching)

**Challenge:**
When two agents attempt to establish a P2P WireGuard tunnel, they usually reside behind strictly configured NATs and residential routers. Direct communication drops because their `192.168.x.x` IPs are non-routable over the internet.

**Solution:**
Instead of relaying all traffic through a central proxy, we integrated the **libp2p** networking stack:
- **AutoNAT & STUN:** The agent discovers its true public IP and port from the perspective of the outside world.
- **Hole Punching:** `libp2p.EnableHolePunching()` coordinates simultaneous connection attempts from both peers. By running a libp2p Ping *before* configuring the WireGuard tunnel, we punch a hole through the NAT — WireGuard UDP packets flow directly (true P2P, no relay).

### 2. Centralized vs. Decentralized State

**Challenge:**
If the central Discovery Service dies, newly joined agents can't find peers — a single point of failure.

**Solution (Hybrid Architecture):**
Gordion uses a **Hybrid Control Plane**. Centralized microservices handle authentication, IP allocation, and act as bootstrap nodes. Once the initial peer list is acquired, agents communicate *directly* via libp2p. The control plane is only needed for periodic heartbeats and new peer discovery — not for ongoing data transfer.

### 3. Per-Peer Packet Routing

**Challenge:**
An initial bridge design would broadcast every WireGuard packet to all connected peers. With 10 peers, each packet is sent 10 times — 9 copies wasted.

**Solution:**
Each peer is assigned a **dedicated local UDP proxy port**. WireGuard's peer endpoint points to the specific proxy, so packets are routed 1:1 to the correct libp2p stream. No broadcast, no wasted bandwidth.

### 4. Exit Node Without a Double-Encryption Penalty

**Challenge:**
Routing through an exit node risks encrypting traffic twice (WireGuard on top of WireGuard) or breaking the NAT bridge.

**Solution:**
The exit node peer uses the same libp2p bridge as any other peer. The only difference is `AllowedIPs = 0.0.0.0/0` on the client side, and `iptables MASQUERADE` on the exit node side. WireGuard still handles encryption once — no extra overhead.

## License

MIT License — see [LICENSE](LICENSE) file for details.
