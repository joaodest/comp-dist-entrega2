# Architecture Update Against the Original Plan

**Date:** 2026-05-01  
**Scope:** what was implemented after the monorepo/containerization plan.

## Summary

The original goal was to evaluate the reverted implementation and reorganize it into a monorepo with separately containerized Gateway, Lobby and Game services. That restructuring was implemented at the repository root. The old nested `voxel-royale/` layout is no longer the active structure.

Because Lobby was requested as boilerplate only, the implemented runtime proof is not the planned room flow yet. The verified path is:

```text
HTTP client
  -> Gateway service (:8080)
  -> Game service gRPC (game:50051)
  -> authoritative gameplay snapshot
```

Lobby is present as a separate service/container, but it does not yet own durable room state or trigger Game.

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
| Gateway service | Implemented as HTTP entrypoint with healthcheck and grpc-gateway proxy to Game. |
| Game service | Implemented as separate gRPC service with authoritative `StreamMatch` movement, chests, weapons, damage, safe zone and ranking behavior. |
| Lobby service | Implemented as separate boilerplate service with gRPC stubs and healthcheck. |
| Per-service containers | Implemented with one Dockerfile per service. |
| Docker Compose | Implemented in `deployments/docker-compose.yml`; Gateway depends on healthy Game and Lobby. |
| Go version | Updated to Go 1.25.0 to support the selected grpc-gateway dependency. |
| Gateway -> Lobby -> Game room flow | Deferred because Lobby was intentionally limited to boilerplate. |

## Current Service Contracts

| Service | Runtime role | Exposed interface |
| --- | --- | --- |
| Gateway | Public HTTP edge | `GET /healthz`, `POST /v1/match/stream` |
| Game | Authoritative gameplay backend | gRPC `GameService.StreamMatch`, health `GET /healthz` on `:8082` |
| Lobby | Boilerplate service boundary | gRPC `CreateRoom`, `JoinRoom`, health `GET /healthz` on `:8081` |

## Remaining Gaps

- Replace Lobby boilerplate with real room lifecycle: create, join, inspect, ready and start.
- Add the planned Gateway -> Lobby -> Game flow once Lobby owns room state.
- Split `StreamMatch` into clearer match lifecycle contracts if the team wants explicit `StartMatch`, input and snapshot streams.
- Add WebSocket real-time input/snapshot pipeline.
- Add observability, correlated request IDs, stress testing and failure handling.
- Regenerate protobuf files from local `protoc` once the team standardizes the toolchain.

## Validation Evidence

The implementation was validated with:

```powershell
go test ./...
docker compose -f deployments/docker-compose.yml config
docker compose -f deployments/docker-compose.yml build
docker compose -f deployments/docker-compose.yml up -d
curl http://localhost:8080/healthz
curl -X POST http://localhost:8080/v1/match/stream -H 'Content-Type: application/json' -d '{"playerId":"player-1","moveX":1,"moveY":2,"inputSequence":1,"isAttacking":false}'
docker compose -f deployments/docker-compose.yml down
```

Observed smoke response:

```json
{"tick":"1","players":[{"playerId":"player-1","x":1,"y":2,"isAlive":true,"health":100,"weapon":"pistol"}],"safeZone":{"centerX":0,"centerY":0,"radius":44.866665,"phase":"0"},"remainingTicks":"299"}
```
