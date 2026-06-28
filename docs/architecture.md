# Architecture Update Against the Original Plan

**Date:** 2026-05-01 (updated 2026-05-09)  
**Scope:** what was implemented after the monorepo/containerization plan.

## Summary

The original goal was to evaluate the reverted implementation and reorganize it into a monorepo with separately containerized Gateway, Lobby and Game services. That restructuring was implemented at the repository root. The old nested `voxel-royale/` layout is no longer the active structure.

Two runtime paths are now verified:

```text
Room flow:
  HTTP client
    -> Gateway service (:8080)
    -> Lobby service gRPC (lobby:50052)
    -> room lifecycle (create, join, get, start, leave)

Gameplay flow:
  HTTP client
    -> Gateway service (:8080)
    -> Game service gRPC (game:50051)
    -> authoritative gameplay snapshot
```

Lobby is fully implemented with in-memory room state and connected to the Gateway via grpc-gateway.

## Implemented Structure

```text
.
├── deployments/docker-compose.yml
├── gen/
│   ├── lobby/
│   └── match/
├── internal/
│   ├── gateway/
│   ├── game/
│   └── lobby/
├── proto/
│   ├── lobby/v1/lobby.proto
│   └── match/v1/match.proto
└── services/
    ├── gateway/
    ├── game/
    └── lobby/
```

## Plan Delta

| Planned item | Implemented result |
| --- | --- |
| Monorepo structure | Implemented at repo root with service, internal, proto, gen and deployment boundaries. |
| Gateway service | Implemented as HTTP entrypoint with healthcheck and grpc-gateway proxy to Game and Lobby. |
| Game service | Implemented as separate gRPC service with authoritative room-scoped gameplay: server-clock movement, chests, weapons, damage, safe zone and ranking behavior. |
| Lobby service | Implemented with in-memory room state: CreateRoom, JoinRoom, GetRoom, StartRoom, LeaveRoom. 24 unit tests. |
| Per-service containers | Implemented with one Dockerfile per service. |
| Docker Compose | Implemented in `deployments/docker-compose.yml`; Gateway depends on healthy Game and Lobby. |
| Go version | Updated to Go 1.25.0 to support the selected grpc-gateway dependency. |
| Gateway -> Lobby room flow | Implemented. Gateway proxies HTTP to Lobby gRPC via grpc-gateway with HTTP annotations. |

## Current Service Contracts

| Service | Runtime role | Exposed interface |
| --- | --- | --- |
| Gateway | Public HTTP edge | `GET /healthz`, WebSocket `GET /v1/match/ws`, `POST /v1/match/stream`, `POST /v1/rooms`, `POST /v1/rooms/{id}/join`, `GET /v1/rooms/{id}`, `POST /v1/rooms/{id}/start`, `POST /v1/rooms/{id}/leave` |
| Game | Authoritative gameplay backend | gRPC `GameService.StreamMatch`, `StartMatch`, `PushInput`, `WatchMatch` (server-clock snapshot stream), health `GET /healthz` on `:8082` |
| Lobby | Room lifecycle manager | gRPC `LobbyService.CreateRoom`, `JoinRoom`, `GetRoom`, `StartRoom`, `LeaveRoom`, health `GET /healthz` on `:8081` |

## Realtime Pipeline (Phase 4)

- Browser keeps a WebSocket (`/v1/match/ws?room&player`) open for the whole match.
- Client inputs (JSON `PlayerInput`) are forwarded to Game via gRPC `PushInput`.
- Game runs a per-room **server clock** (~15 Hz) that advances the authoritative
  simulation from buffered inputs, independent of client cadence.
- Game streams a `GameState` snapshot per tick via `WatchMatch`; the Gateway
  fans out each snapshot to every connected WebSocket of that room.
- The clock starts on the first `WatchMatch` subscriber and stops when the last
  one disconnects, so unwatched matches consume no CPU. This replaced the Phase 1
  unary auto-restart stopgap for room matches.

## Remaining Gaps

- Add observability, correlated request IDs, stress testing and failure handling.
- Add structured logging with request_id, room_id, player_id across services.
- Robust reconnection and disconnect handling that keeps matches alive (Phase 7).

## Validation Evidence

The implementation was validated with:

```bash
go test ./...
docker-compose -f deployments/docker-compose.yml build
docker-compose -f deployments/docker-compose.yml up -d
curl http://localhost:8080/healthz
curl -X POST http://localhost:8080/v1/match/stream -H 'Content-Type: application/json' -d '{"playerId":"player-1","moveX":1,"moveY":2,"inputSequence":1,"isAttacking":false}'
curl -X POST http://localhost:8080/v1/rooms -H 'Content-Type: application/json' -d '{"ownerName":"Ana","maxPlayers":10}'
curl -X POST http://localhost:8080/v1/rooms/room-1/join -H 'Content-Type: application/json' -d '{"playerName":"Bruno"}'
curl http://localhost:8080/v1/rooms/room-1
curl -X POST http://localhost:8080/v1/rooms/room-1/start -H 'Content-Type: application/json' -d '{"playerId":"player-1-1"}'
docker-compose -f deployments/docker-compose.yml down
```

Observed smoke responses:

Game:
```json
{"tick":"1","players":[{"playerId":"player-1","x":1,"y":2,"isAlive":true,"health":100,"weapon":"pistol"}],"safeZone":{"centerX":0,"centerY":0,"radius":44.991112,"phase":"0"},"remainingTicks":"4499"}
```

Lobby:
```json
{"roomId":"room-1","status":"ROOM_STATUS_WAITING","ownerId":"player-1-1","players":[{"playerId":"player-1-1","playerName":"Ana"},{"playerId":"player-room-1-2","playerName":"Bruno"}],"maxPlayers":10,"joinUrl":"/v1/rooms/room-1/join"}
```
