# Implementation Delta

This document records what changed relative to the original plan for the monorepo/containerization task.

## Original Plan

The approved plan asked to:

- evaluate the reverted implementation and repository gaps;
- reorganize the repo as a monorepo;
- segregate Gateway, Lobby and Game;
- containerize each service independently;
- keep Lobby as boilerplate;
- preserve the implemented Gateway/Game behavior where possible;
- update Go if required.

## Completed

- Created root Go module `voxel-royale` with Go `1.25.0`.
- Restored generated gRPC/grpc-gateway code from the reverted branch under `gen/`.
- Added versioned contracts under `proto/lobby/v1` and `proto/match/v1`.
- Added `internal/gateway`, `internal/game` and `internal/lobby` packages.
- Added `services/gateway`, `services/game` and `services/lobby` process entrypoints.
- Added one Dockerfile per service.
- Added `deployments/docker-compose.yml` with healthchecks and service DNS.
- Added Makefile targets for test, proto, Compose build/up/down and demo output.
- Added README runbook and smoke-test commands.

## Behavior Implemented

- Gateway exposes HTTP on `:8080`.
- Gateway proxies `POST /v1/match/stream` to Game through gRPC.
- Gateway proxies Lobby room endpoints (`/v1/rooms/*`) to Lobby through gRPC via grpc-gateway.
- Game validates `player_id` and computes an in-memory authoritative gameplay snapshot with movement, chest opening, weapon pickup, damage, elimination, safe zone and ranking.
- Lobby manages room lifecycle with in-memory state: CreateRoom, JoinRoom, GetRoom, StartRoom, LeaveRoom. Includes owner validation, capacity limits, ownership transfer and room closure.

## Deviations

- The original planning docs expected a room flow using Lobby. This was not implemented because the request explicitly scoped Lobby to boilerplate.
- `gen/` is used for generated code instead of `internal/contracts/`, matching the restored code and keeping imports stable.
- The implemented gameplay-facing RPC remains `StreamMatch`, not `StartMatch`; it now carries gameplay input and snapshot fields beyond the initial skeleton.

## Remaining Work

- Connect Lobby StartRoom to Game service for match start.
- Add request correlation/logging across services (slog with request_id, room_id, player_id).
- Add WebSocket gameplay transport.
- Tune gameplay balance and replace the single in-memory match with room-scoped match state.
- Add observability stack and 50-player load runner.
