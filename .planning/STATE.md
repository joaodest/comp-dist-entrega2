# Project State: Voxel Royale Distribuido

## Project Reference

See: `.planning/PROJECT.md` (updated 2026-04-24)

**Core value:** Demonstrar, de forma jogavel e mensuravel, um sistema distribuido em tempo real no qual 50 jogadores participam de uma partida battle royale voxel com backend Go autoritativo e comunicacao entre servicos via gRPC.  
**Current focus:** Phase 1 - Entrega 1 Distributed Skeleton (code complete; report draft pending)

## Current Position

**Phase:** 1  
**Plan:** 01-01..01-04 executed; 01-05 partial  
**Status:** Distributed skeleton implemented and tested; Entrega 1 SBC report draft still missing  
**Progress:** ██░░░░░░░░ ~15%

## Performance Metrics

- Requirements total: 40
- Requirements mapped: 40
- Phases total: 8
- Phase 1 plans complete: 4/5 (01-05 partial)
- Current delivery target: Entrega 1

## Accumulated Context

### Decisions

- Backend: Go (`go 1.25.0`, module `voxel-royale`).
- Internal communication: gRPC/Protocol Buffers.
- Public web services: HTTP APIs exposed by Gateway via grpc-gateway (`/v1/rooms/*`, `/v1/match/stream`).
- Realtime browser transport: WebSocket (planned, not yet implemented).
- Frontend: Babylon.js + TypeScript (planned).
- Infrastructure: Docker Compose, one Dockerfile per service, portable local/VPS deployment.
- Entrega 1 mandatory requirements: gRPC/RPC + web services.
- Generated code lives under `gen/` (not `internal/contracts/`), matching the restored branch.
- Versioned contracts under `proto/lobby/v1` and `proto/match/v1`.
- Gameplay RPC is `StreamMatch` (carries input + snapshot), not `StartMatch`.
- Lobby supports CreateRoom, JoinRoom, GetRoom, StartRoom, LeaveRoom and SetReady (player ready state, auto-starts when all ready). PR #6 merged 2026-06-26.

### Todos

- Finish plan 01-05: write `docs/messages.md`, `docs/roles.md` and the Entrega 1 SBC report draft (`docs/report/`).
- Fill student names and ownership roles.
- Confirm Canvas dates for Entrega 1 and Entrega 2.
- Validate Hostinger VPS resources before deploy phase.
- Connect `Lobby.StartRoom` to the Game service for match start.
- Add request correlation/logging across services (request_id, room_id, player_id).
- Plan Phase 2 from the roadmap once Entrega 1 report is closed.

### Blockers

- Entrega 1 report draft not written yet (blocks Phase 1 success criterion #5).
- Student roster/roles not documented yet.

## Session Continuity

Phase 1 is functionally implemented (Gateway, Lobby, Game build and pass tests; Docker Compose with healthchecks).
The remaining Phase 1 work is documentation: messages reference, roles and the SBC report draft (plan 01-05).

Next recommended command:

```text
$gsd-plan-phase 1 --resume   # finish plan 01-05 (docs + report)
```

Then proceed to Phase 2 (Team Development System).

---
*State updated: 2026-06-26 after merging PR #6 (player ready state) and reconciling state with implemented code*
