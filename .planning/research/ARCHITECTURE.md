# Architecture Patterns

**Domain:** distributed real-time browser game  
**Researched:** 2026-04-24  
**Overall confidence:** MEDIUM

## Implementation Update - 2026-05-01

This section records what was implemented against the original monorepo/containerization plan. It is not a replacement for the long-term target architecture below.

### What Was Implemented

| Original plan item | Current implementation | Status |
| --- | --- | --- |
| Restructure repo as a monorepo instead of keeping code under `voxel-royale/`. | Go module moved to repo root with `services/`, `internal/`, `proto/`, `gen/` and `deployments/`. | Done |
| Containerize Gateway, Lobby and Game independently. | Each service has its own Dockerfile under `services/<service>/Dockerfile`; Compose starts `gateway`, `lobby` and `game`. | Done |
| Keep Gateway as the public HTTP entrypoint. | `services/gateway` exposes `GET /healthz` and the grpc-gateway route `POST /v1/match/stream`. | Done |
| Keep Game as a separate gRPC service. | `services/game` exposes gRPC on `:50051`, health on `:8082`, and implements `GameService.StreamMatch` with authoritative gameplay state. | Done |
| Add Lobby as boilerplate only. | `services/lobby` exposes gRPC on `:50052`, health on `:8081`, and has stub CreateRoom/JoinRoom behavior marked as boilerplate. | Done |
| Avoid container-localhost coupling. | Compose injects `GAME_GRPC_ADDR=game:50051` into Gateway. | Done |
| Update Go version if needed. | `go.mod` now targets Go `1.25.0` because `grpc-gateway/v2.29.0` requires Go 1.25+. | Done |

### Deliberate Deviations From the Original Plan

- The original Phase 1 plan expected Gateway -> Lobby -> Game for room creation/start. The user explicitly asked for Lobby as boilerplate, so the implemented smoke path is Gateway -> Game through `StreamMatch`.
- Generated code currently lives under `gen/` to match the restored contracts from the reverted branch. The earlier plan mentioned `internal/contracts/...`; that structure was not used in this implementation.
- The implemented Game method is `StreamMatch`, inherited from the restored `match.proto`; the earlier planning docs mentioned `StartMatch` as the future room-start contract.
- Observability, WebSocket, QR lobby, full room lifecycle, explicit match start, stress tests and fault-tolerance behavior remain future work.

### Verified Commands

```powershell
go test ./...
docker compose -f deployments/docker-compose.yml config
docker compose -f deployments/docker-compose.yml build
docker compose -f deployments/docker-compose.yml up -d
curl http://localhost:8080/healthz
curl -X POST http://localhost:8080/v1/match/stream -H 'Content-Type: application/json' -d '{"playerId":"player-1","moveX":1,"moveY":2,"isAttacking":false}'
docker compose -f deployments/docker-compose.yml down
```

The smoke response confirmed HTTP Gateway -> gRPC Game:

```json
{"tick":"1","players":[{"playerId":"player-1","x":1,"y":2,"isAlive":true,"health":100,"weapon":"pistol"}],"remainingTicks":"299"}
```

## Recommended Architecture

The system should use a small set of Go services with explicit contracts:

- Browser client connects to a public Gateway over HTTP/WebSocket.
- Gateway exposes web services and keeps WebSocket sessions.
- Lobby service manages rooms, QR tokens, player names and ready state.
- Game service owns authoritative match state and runs the tick loop.
- Telemetry stack collects traces, metrics and logs.
- Bot/load runner simulates 50 players for stress tests.

```text
Mobile Browser
  | HTTP/WebSocket
Gateway Service
  | gRPC
Lobby Service ---- gRPC ---- Game Service
  |                         |
  | metrics/traces          | metrics/traces
  v                         v
Prometheus / Jaeger / Grafana
```

## Component Boundaries


| Component       | Responsibility                                                                | Communicates With                |
| --------------- | ----------------------------------------------------------------------------- | -------------------------------- |
| Frontend client | 3D render, touch controls, local prediction/interpolation, QR join flow.      | Gateway HTTP/WebSocket           |
| Gateway service | Public entrypoint, WebSocket sessions, request validation, fanout to clients. | Lobby and Game via gRPC          |
| Lobby service   | Room lifecycle, player registration, ready state, match start request.        | Gateway and Game via gRPC        |
| Game service    | Authoritative tick, spawn, collision, chests, weapons, safe zone, ranking.    | Gateway and Lobby via gRPC       |
| Telemetry stack | Metrics, traces, dashboards.                                                  | All services via OTel/Prometheus |
| Load simulator  | Simulated players for 50-player stress tests.                                 | Gateway HTTP/WebSocket           |


## Main Messages


| Message       | Direction                         | Content                                                |
| ------------- | --------------------------------- | ------------------------------------------------------ |
| CreateRoom    | HTTP -> Gateway -> Lobby gRPC     | Room settings, max players, match duration.            |
| JoinRoom      | HTTP -> Gateway -> Lobby gRPC     | Room token, player display name.                       |
| StartMatch    | Lobby gRPC -> Game gRPC           | Room id, players, match config.                        |
| PlayerInput   | WebSocket -> Gateway -> Game gRPC | Player id, input sequence, movement/action commands.   |
| StateSnapshot | Game gRPC -> Gateway -> WebSocket | Tick id, player transforms, health, chests, safe zone. |
| PlayerAction  | WebSocket -> Gateway -> Game gRPC | Attack/open chest/use weapon command.                  |
| MatchEnded    | Game gRPC -> Lobby/Gateway        | Ranking, eliminations, duration, final state.          |


## Patterns to Follow

### Server-authoritative simulation

The Game service is the only authority for damage, eliminations, chest contents and safe-zone timing. Clients may predict movement for feel, but must reconcile against server snapshots.

### Contract-first service development

Define `.proto` contracts before service implementation. Every team implements against generated interfaces, not improvised JSON.

### Stateless public services

Gateway and Lobby should be restartable. Match state can live in Game service memory for v1, but service boundaries must make future replication/failover possible.

### Observable by default

Every gRPC call and HTTP/WebSocket lifecycle should emit trace/span metadata. Tick duration, connected players and snapshot sizes are first-class metrics.

## Anti-Patterns to Avoid

### Browser-authoritative gameplay

If clients decide hits, damage or inventory, the game diverges and the distributed-system story weakens.

### One monolithic Go process

It may be easier, but it fails the architecture demonstration. Keep at least Gateway, Lobby and Game as separate services.

### Hidden manual setup

The professor and teammates should be able to run the system from documented Docker Compose commands.

## Scalability Considerations


| Concern        | 50 players                                   | Next step                         | Long-term                                 |
| -------------- | -------------------------------------------- | --------------------------------- | ----------------------------------------- |
| Game tick      | One Game service instance can own one match. | One process hosts multiple rooms. | Shard matches across game workers.        |
| Gateway fanout | One gateway can hold WebSockets.             | Add sticky room routing.          | External session routing/load balancer.   |
| State size     | Send compact snapshots.                      | Delta compression.                | Interest management/spatial partitioning. |
| Observability  | Local Prometheus/Jaeger.                     | VPS dashboards.                   | Centralized logs and alerts.              |


## Sources

- Course instructions: `docs/course-instructions.md`
- User PRD and architecture decisions.
