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

### Observability

| Tool | Port | Status | Description |
|------|------|--------|-------------|
| **Prometheus** | 9091 | Complete | Metrics collection from all services |
| **Grafana** | 3000 | Complete | Metrics visualization and dashboards |
| **Distributed Tracing** | - | Complete | Cross-service trace_id propagation |
| **Structured Logging** | - | Complete | Request-scoped logging with request_id |
| **Rate Limiting** | - | Complete | Per-IP sliding window request limiting |
| **Health Checks** | - | Complete | gRPC standard health checking protocol |

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
│       │   └── wireguard/     # Tunnel management, key generation
│       ├── Dockerfile
│       └── test/              # End-to-end integration tests
├── pkg/
│   ├── auth/                  # Inter-service authentication client
│   ├── config/                # Configuration management
│   ├── grpcutil/              # gRPC error utilities
│   ├── logger/                # Structured logging (zerolog)
│   ├── metrics/               # Prometheus metrics
│   ├── middleware/            # Logging interceptor (request_id)
│   ├── tlsutil/               # TLS credential helpers
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
make docker-up

# Run database migrations
docker exec -i gordion-postgres psql -U gordion -d gordion \
  < services/identity/migrations/0001_initial.sql

# Build all services
make build-all

# Start services (each in a separate terminal)
cd services/identity  && ./identity-server.exe
cd services/discovery && ./discovery-server.exe
cd services/config    && ./config-server.exe

# Start the agent
make build-agent
./services/agent/agent.exe
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

| Node Authentication | JWT tokens via Identity Service |
| Inter-Service Auth | Token validation between services |
| Transport Security | Optional TLS for gRPC (cert generation via `scripts/gen-certs.ps1`) |
| Tunnel Encryption | WireGuard with Curve25519 key exchange |
| Abuse Prevention | Per-IP Rate Limiting (100 req/min default) |
| Secret Management | Environment variable overrides, `.env` support |

## Agent Lifecycle

```
Start:
  1. Generate WireGuard keypair (Curve25519)
  2. Register with Identity Service (with **Exponential Backoff**) → get token
  3. Fetch network config (supports **Config Versioning**)
  4. Request VPN IP address
  5. Announce to Discovery Service
  6. Discover other peers
  7. Fetch peer public keys from Identity Service
  8. Configure WireGuard tunnel (real tunnel or dry-run)
  9. Start background loops:
     - **Heartbeat Loop**: Keeps peer status alive
     - **Token Refresh Loop**: Automatically re-registers at 80% of token life

Shutdown (Ctrl+C):
  1. **Graceful Shutdown**: Wait for in-flight requests (10s timeout)
  2. Stop heartbeat & refresh loops
  3. Release VPN IP from Config Service
  4. Tear down WireGuard tunnel
  5. Close all gRPC connections
```

## Observability

### Metrics

All services expose Prometheus metrics via dedicated HTTP endpoints.

| Service | Metrics Endpoint |
|---------|-----------------|
| Identity | `http://localhost:9090/metrics` |
| Discovery | `http://localhost:9091/metrics` |
| Config | `http://localhost:9092/metrics` |
| Grafana | `http://localhost:3000` (admin / admin) |
| Prometheus | `http://localhost:9091` |

| Metric | Description |
|--------|-------------|
| `gordion_grpc_requests_total` | Total gRPC request count by service, method, status |
| `gordion_grpc_request_duration_seconds` | Request latency histogram |
| `gordion_active_connections` | Current active connections per service |
| `gordion_db_queries_total` | Database query count |
| `gordion_db_query_duration_seconds` | Database query latency |

### Distributed Tracing

Trace IDs propagate across services via gRPC metadata (`x-trace-id` header). A single request can be tracked across all services using the same trace ID.

```
Agent [trace: a3f29b01] → Identity [trace: a3f29b01] → Config [trace: a3f29b01]
```

### Structured Logging

Each request is assigned a unique `request_id` via the logging interceptor. Combined with `trace_id`, this provides full request visibility:

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
| `RegisterNode` | `public_key`, `version` | `node_id`, `token`, `expires_at` |
| `ValidateToken` | `token` | `valid`, `node_id` |
| `GetPublicKey` | `node_id` | `public_key` |

### Discovery Service (port 8002)

| Method | Request | Response |
|--------|---------|----------|
| `RegisterPeer` | `token`, `ip_address`, `port`, `region` | `success`, `message` |
| `ListPeers` | `region`, `limit` | `peers[]` |
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

### Sprint 1: Foundation - Complete
- Monorepo setup, proto definitions, shared packages

### Sprint 2: Identity Service - Complete
- PostgreSQL storage, JWT authentication, gRPC API, integration tests, Prometheus metrics

### Sprint 3: Discovery Service - Complete
- Redis registry, peer matching, heartbeat mechanism, gRPC API, integration tests, Prometheus metrics

### Sprint 4: Config Service - Complete
- IP allocation (DHCP-like), network configuration, gRPC API, integration tests, Prometheus metrics

### Sprint 5: Agent - Complete
- WireGuard tunnel management, Curve25519 key generation, full lifecycle orchestration, end-to-end tests

### Sprint 6: Security & Observability - Complete
- Inter-service authentication, optional TLS, secret management, structured logging, distributed tracing

### Sprint 7: Resilience & Polish - Complete
- gRPC Health Check protocol integration
- Per-IP Rate Limiting (sliding window)
- Agent: Exponential backoff retries & Token refresh loop
- Config: SIGHUP hot-reload & Version-based caching
- Global: Graceful shutdown with timeout (10s)
- Makefile improvements & project audit fixes

## License

MIT License - see [LICENSE](LICENSE) file for details.
