# Project State: Voxel Royale Distribuido

## Project Reference

See: `.planning/PROJECT.md` (updated 2026-04-24)

**Core value:** Demonstrar, de forma jogavel e mensuravel, um sistema distribuido em tempo real no qual 50 jogadores participam de uma partida battle royale voxel com backend Go autoritativo e comunicacao entre servicos via gRPC.  
**Current focus:** Entrega 2 — Fase 3 concluída: lobby com QR Code e início de partida integrado (Lobby→Game, partidas por sala). Próximo: Fase 4 (WebSocket/tempo real). (Entrega 1 funcional; faltam nomes reais dos alunos.)

## Current Position

**Phase:** 3  
**Plan:** 01-01..01-05; 02-01 by-design; 03-01 QR lobby + Lobby→Game match start  
**Status:** Distributed skeleton + by-design docs + Fase 3 completa. Lobby chama Game.StartMatch via gRPC; Game mantém partidas por sala (room_id); cliente Phaser tem tela de lobby com QR Code, entrada por nome, ready/start. Faltam nomes reais dos alunos.  
**Progress:** ███░░░░░░░ ~33%

## Performance Metrics

- Requirements total: 40
- Requirements mapped: 40
- Phases total: 8
- Phase 1 plans complete: 5/5 (01-05 done; report draft com nomes a preencher)
- Phase 2 plans complete: 1/1 (`ARCH-06`)
- Phase 3 plans complete: 1/1 (`LOBB-01..03` done, `LOBB-04` partial)
- Current delivery target: Entrega 2

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
- Cliente web implementado em `frontend/` (Phaser 3 + TypeScript + Vite), estilo .io top-down (mapa grama/rio/árvores/pedras, jogador controlável, render do `GameState`). MVP adiantado em relação ao roadmap — valida a direção de arte e o caminho Navegador→Gateway→Game. Modo AO VIVO (`POST /v1/match/stream`) com fallback OFFLINE (mock).
- `playerId` do cliente é único por sessão (evita colisão de identidade e input `stale`, já que o servidor guarda o último `inputSequence` por jogador).
- Backend: `GameService.StreamMatch` agora **auto-reinicia** o match quando ele termina. Antes, o match global encerrado (tick ≥ 300) fazia o servidor ignorar todo input e travar para todos.
- Fase 2 iniciada/consolidada: `CONTRIBUTING.md` define checklist de PR e validações; `docs/team-development.md` define fronteiras dos serviços, ownership, processo de contrato, testes e divisão de tarefas; `docs/roles.md` foi expandido com ownership de frontend, observabilidade, carga e deploy.
- Fase 3 concluída: contrato `GameService.StartMatch` (gRPC interno) + `room_id` no `PlayerInput`; Game passou a manter partidas por sala (`matches map[string]*matchState`, chave `room_id`, mais a global `__global__`); Lobby chama o Game ao iniciar (start do dono e auto-start do `SetReady`), fora do lock e com revert para `WAITING` em falha; `services/lobby` disca `GAME_GRPC_ADDR` e o Compose adiciona `depends_on: game`; cliente Phaser ganhou `lobby.ts`/`lobbyUI.ts`/`session.ts` (criar/entrar/QR/ready/start) com dep `qrcode`. Validado ponta a ponta via curl (start cria partida da sala com roster; global isolado).

### Todos

- Fill student names and ownership roles (substituir `PLACEHOLDER` em `docs/roles.md` e `docs/report/entrega1.tex`).
- (Para submissão final) trocar o rascunho `article` pelo `sbc-template` oficial da SBC e re-verificar o limite de 4 páginas.
- Confirm Canvas dates for Entrega 1 and Entrega 2.
- Validate Hostinger VPS resources before deploy phase.
- [x] Connect `Lobby.StartRoom` to the Game service for match start. (Fase 3)
- Add request correlation/logging across services (request_id, room_id, player_id).
- Lobby start-by-time-limit (LOBB-04) e tela de fim de partida/ranking no cliente (Fase 5).
- Use the Phase 2 guide as gate for upcoming work: every task should name requirement, owner, contract impact, validation and docs affected.
- **[ABERTO] Refactor de tempo real (Fase 4):** o `StreamMatch` avança 1 tick por request (modelo unário), então a partida atinge `maxMatchTicks` (300) em ~27s e auto-reinicia (zona/mundo resetam). Trocar para o servidor avançar ticks no **próprio relógio** + transporte **WebSocket** (snapshots em tempo real, desacoplados do request). Fix temporário já aplicado: auto-restart do match encerrado para não travar o input.

### Blockers

- Student roster not filled: `docs/roles.md` e o relatório usam `PLACEHOLDER`. Squads/ownership já estão definidos; faltam os nomes reais (tarefa do grupo).

## Session Continuity

Phase 1 is functionally implemented (Gateway, Lobby, Game build and pass tests; Docker Compose with healthchecks)
and documented: `docs/architecture.md`, `docs/messages.md`, `docs/roles.md` e o relatório SBC draft.
Phase 2 has started with `CONTRIBUTING.md` and `docs/team-development.md`, plus expanded ownership in `docs/roles.md`.
O pendente administrativo ainda é preencher os nomes reais dos alunos (`PLACEHOLDER`).

Next recommended command:

```text
Start Phase 4 (Realtime Network Pipeline): WebSocket + server clock para snapshots em tempo real.
```

Fase 4 vai substituir o atual polling unário (`StreamMatch` 1 tick por request) por WebSocket + relógio
do servidor, removendo o auto-restart como stopgap. A integração de sala da Fase 3 já fornece `room_id`
para rotear sessões de WebSocket por partida.

---
*State updated: 2026-06-27 — Fase 3 concluída (lobby com QR Code + início de partida Lobby→Game, partidas por sala). TODO aberto: preencher nomes reais e refactor de tempo real (WebSocket, Fase 4).*
