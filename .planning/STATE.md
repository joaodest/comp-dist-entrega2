# Project State: Voxel Royale Distribuido

## Project Reference

See: `.planning/PROJECT.md` (updated 2026-04-24)

**Core value:** Demonstrar, de forma jogavel e mensuravel, um sistema distribuido em tempo real no qual 50 jogadores participam de uma partida battle royale voxel com backend Go autoritativo e comunicacao entre servicos via gRPC.  
**Current focus:** Phase 1 - Entrega 1 Distributed Skeleton (code + docs/report draft complete; pending real student names)

## Current Position

**Phase:** 1  
**Plan:** 01-01..01-05 executed  
**Status:** Distributed skeleton implemented/tested; Entrega 1 docs (messages, roles) and SBC report draft (entrega1.pdf, 4 páginas) gerados. Faltam apenas os nomes reais dos alunos.  
**Progress:** ██░░░░░░░░ ~18%

## Performance Metrics

- Requirements total: 40
- Requirements mapped: 40
- Phases total: 8
- Phase 1 plans complete: 5/5 (01-05 done; report draft com nomes a preencher)
- Current delivery target: Entrega 1

## Accumulated Context

### Decisions

- Backend: Go (`go 1.25.0`, module `voxel-royale`).
- Internal communication: gRPC/Protocol Buffers.
- Public web services: HTTP APIs exposed by Gateway via grpc-gateway (`/v1/rooms/*`, `/v1/match/stream`).
- Realtime browser transport: WebSocket (planned, not yet implemented).
- Frontend: Phaser (2D) + TypeScript (planned). Trocado de Babylon.js (3D) para Phaser (2D) em 2026-06-27 por facilidade; o backend ja trabalha em coordenadas 2D (x/y).
- Infrastructure: Docker Compose, one Dockerfile per service, portable local/VPS deployment.
- Entrega 1 mandatory requirements: gRPC/RPC + web services.
- Generated code lives under `gen/` (not `internal/contracts/`), matching the restored branch.
- Versioned contracts under `proto/lobby/v1` and `proto/match/v1`.
- Gameplay RPC is `StreamMatch` (carries input + snapshot), not `StartMatch`.
- Lobby supports CreateRoom, JoinRoom, GetRoom, StartRoom, LeaveRoom and SetReady (player ready state, auto-starts when all ready). PR #6 merged 2026-06-26.
- Entrega 1 docs criados refletindo os contratos REAIS (não os nomes supostos no plano 01-05): `docs/messages.md` documenta `match.proto`/`StreamMatch` (não `game.proto`/`StartMatch`); `docs/roles.md` define 3 squads para 9 alunos.
- Relatório SBC é um rascunho em classe `article` (portátil) que compila com MiKTeX/pdflatex via `docs/report/build.ps1`; trocar pelo `sbc-template` oficial na submissão final. PDF atual tem exatamente 4 páginas (no limite).

### Todos

- Fill student names and ownership roles (substituir `PLACEHOLDER` em `docs/roles.md` e `docs/report/entrega1.tex`).
- (Para submissão final) trocar o rascunho `article` pelo `sbc-template` oficial da SBC e re-verificar o limite de 4 páginas.
- Confirm Canvas dates for Entrega 1 and Entrega 2.
- Validate Hostinger VPS resources before deploy phase.
- Connect `Lobby.StartRoom` to the Game service for match start.
- Add request correlation/logging across services (request_id, room_id, player_id).
- Plan Phase 2 from the roadmap once Entrega 1 report is closed.

### Blockers

- Student roster not filled: `docs/roles.md` e o relatório usam `PLACEHOLDER`. Squads/ownership já estão definidos; faltam os nomes reais (tarefa do grupo).

## Session Continuity

Phase 1 is functionally implemented (Gateway, Lobby, Game build and pass tests; Docker Compose with healthchecks)
and now documented: `docs/architecture.md`, `docs/messages.md`, `docs/roles.md` e o relatório SBC draft
(`docs/report/entrega1.tex` → `entrega1.pdf`, 4 páginas). Plan 01-05 concluído. O único pendente da Fase 1 é
preencher os nomes reais dos alunos (`PLACEHOLDER`).

Next recommended command:

```text
$gsd-plan-phase 2            # Team Development System (by-design guide para 9 alunos)
```

Antes da Fase 2, vale reconciliar o ROADMAP: o ready-state do Lobby (LOBB-03, escopo da Fase 3) já foi
implementado na Fase 1.

---
*State updated: 2026-06-27 after completing plan 01-05 (Entrega 1 docs + SBC report draft)*
