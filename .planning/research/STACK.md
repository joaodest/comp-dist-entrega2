# Technology Stack

**Project:** Voxel Royale Distribuido  
**Researched:** 2026-04-24  
**Overall confidence:** MEDIUM - stack choices are aligned with project goals and official ecosystem positioning, but exact package versions should be pinned during implementation.

## Recommended Stack

### Core Backend

| Technology | Version Policy | Purpose | Why |
|------------|----------------|---------|-----|
| Go | Pin current stable in `go.mod` | Backend services | Strong concurrency model, low deployment overhead, mature gRPC support. |
| gRPC + Protocol Buffers | Pin via Go modules | Internal service contracts | Satisfies Entrega 1 RPC requirement and creates typed contracts between services. |
| REST/HTTP web services | Go standard library or lightweight router | Lobby/status/admin APIs | Satisfies Entrega 1 web services requirement and supports easy demos. |
| WebSocket | Go gateway service | Real-time client connection | Simpler than WebRTC for v1 and enough for input/snapshot exchange. |

### Frontend

| Technology | Version Policy | Purpose | Why |
|------------|----------------|---------|-----|
| TypeScript | Pin in package.json | Client app | Safer contracts for network payloads and 2D game code. |
| Phaser | Pin in package.json | 2D rendering | Mature browser 2D game engine with mobile-friendly input; simpler than a 3D engine for the academic scope and matches the already-2D backend (x/y). |
| Vite | Pin in package.json | Frontend dev/build | Fast student-friendly workflow and simple static deployment. |

### Observability and Testing

| Technology | Version Policy | Purpose | Why |
|------------|----------------|---------|-----|
| OpenTelemetry | Pin Go SDK modules | Traces and metrics instrumentation | Makes gRPC flows and distributed behavior visible. |
| Prometheus | Docker image tag pinned | Metrics scraping | Standard for tick rate, latency, load and service health. |
| Grafana | Docker image tag pinned | Dashboards | Clear visual demo for evaluation. |
| Jaeger | Docker image tag pinned | Trace visualization | Shows request flow across gateway, lobby and game services. |
| k6 or custom Go bots | Pin tool/image | Stress tests | Simulates 50 players and proves scalability claims. |

### Infrastructure

| Technology | Version Policy | Purpose | Why |
|------------|----------------|---------|-----|
| Docker | Stable local install | Containerization | Portable grading and development environment. |
| Docker Compose | Compose file versionless spec | Local and VPS orchestration | Fits the requirement for stateless transportable infrastructure. |
| Nginx or Caddy | Pinned image if needed | Reverse proxy/TLS on VPS | Provides single public entrypoint for frontend, API and WebSocket. |

## Alternatives Considered

| Category | Recommended | Alternative | Why Not |
|----------|-------------|-------------|---------|
| Realtime transport | WebSocket | WebRTC | WebRTC adds NAT/signaling complexity that does not help Entrega 1. |
| Client game engine | Phaser (2D) | Babylon.js / Three.js (3D) | 3D engines add modeling/rendering complexity; Phaser 2D matches the already-2D backend (x/y) and is faster for the team to learn. |
| Backend language | Go | Node.js | Go better demonstrates concurrency and gRPC in the discipline context. |
| Orchestration | Docker Compose | Kubernetes | Kubernetes is unnecessary for the first two deliveries and would dilute focus. |

## Implementation Notes

- Keep `.proto` files in a shared contracts package and generate Go clients/servers from them.
- Keep HTTP DTOs and gRPC messages intentionally close, but do not hand-serialize gRPC payloads.
- Prefer stateless services; store active match state only in the owning game service instance.
- Make every service expose `/healthz` and `/readyz` web endpoints.

## Sources

- Course instructions: `docs/course-instructions.md`
- User PRD and stack decisions in initialization conversation.
