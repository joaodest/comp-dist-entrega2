# Project State: Voxel Royale Distribuido

## Project Reference

See: `.planning/PROJECT.md` (updated 2026-04-24)

**Core value:** Demonstrar, de forma jogavel e mensuravel, um sistema distribuido em tempo real no qual 50 jogadores participam de uma partida battle royale voxel com backend Go autoritativo e comunicacao entre servicos via gRPC.  
**Current focus:** Entrega 2 — Fase 5 concluída: loop de partida jogável, zona segura de até 5 minutos e tela final com ranking. Próximo: Fase 6 (observabilidade + prova de 50 jogadores). (Entrega 1 funcional; faltam nomes reais dos alunos.)

## Current Position

**Phase:** 5  
**Plan:** 01-01..01-05; 02-01 by-design; 03-01 QR lobby + Lobby→Game match start; 04-01 realtime pipeline; 05-01 playable loop + final ranking  
**Status:** Distributed skeleton + by-design docs + Fases 3, 4 e 5 completas. Game roda relógio de servidor (15 Hz) por sala com `PushInput`/`WatchMatch`, duração de até 4500 ticks (~5 min), safe zone progressiva, baús/armas/dano/eliminação/ranking; Gateway expõe WebSocket `/v1/match/ws`; cliente Phaser usa modelo push (`RealtimeClient`), controles touch/teclado, auto-alvo para ataque, fallback offline e tela final com ranking. Faltam nomes reais dos alunos.  
**Progress:** ██████░░░░ ~62%

## Performance Metrics

- Requirements total: 40
- Requirements mapped: 40
- Phases total: 8
- Phase 1 plans complete: 5/5 (01-05 done; report draft com nomes a preencher)
- Phase 2 plans complete: 1/1 (`ARCH-06`)
- Phase 3 plans complete: 1/1 (`LOBB-01..03` done, `LOBB-04` partial)
- Phase 4 plans complete: 1/1 (`NETW-01..04` done)
- Phase 5 plans complete: 1/1 (`GAME-01..08` done)
- Current delivery target: Entrega 2

## Accumulated Context

### Decisions

- Backend: Go (`go 1.25.0`, module `voxel-royale`).
- Internal communication: gRPC/Protocol Buffers.
- Public web services: HTTP APIs exposed by Gateway via grpc-gateway (`/v1/rooms/*`, `/v1/match/stream`).
- Realtime browser transport: WebSocket (`/v1/match/ws`) via Gateway.
- Frontend: Phaser (2D) + TypeScript (planned). Trocado de Babylon.js (3D) para Phaser (2D) em 2026-06-27 por facilidade; o backend ja trabalha em coordenadas 2D (x/y).
- Infrastructure: Docker Compose, one Dockerfile per service, portable local/VPS deployment.
- Entrega 1 mandatory requirements: gRPC/RPC + web services.
- Generated code lives under `gen/` (not `internal/contracts/`), matching the restored branch.
- Versioned contracts under `proto/lobby/v1` and `proto/match/v1`.
- Gameplay RPC is `StreamMatch` (carries input + snapshot), not `StartMatch`.
- Lobby supports CreateRoom, JoinRoom, GetRoom, StartRoom, LeaveRoom and SetReady (player ready state, auto-starts when all ready). PR #6 merged 2026-06-26.
- Entrega 1 docs criados refletindo os contratos REAIS (não os nomes supostos no plano 01-05): `docs/messages.md` documenta `match.proto`/`StreamMatch` (não `game.proto`/`StartMatch`); `docs/roles.md` define 3 squads para 9 alunos.
- Relatório SBC é um rascunho em classe `article` (portátil) que compila com MiKTeX/pdflatex via `docs/report/build.ps1`; trocar pelo `sbc-template` oficial na submissão final. PDF atual tem exatamente 4 páginas (no limite).
- Cliente web implementado em `frontend/` (Phaser 3 + TypeScript + Vite), estilo .io top-down (mapa grama/rio/árvores/pedras, jogador controlável, render do `GameState`). Modo AO VIVO usa WebSocket `/v1/match/ws` com fallback OFFLINE (mock).
- `playerId` do cliente é único por sessão (evita colisão de identidade e input `stale`, já que o servidor guarda o último `inputSequence` por jogador).
- Backend: `GameService.StreamMatch` ainda auto-reinicia o match global legado quando termina; partidas de sala usam o relógio da Fase 4/5 e param no snapshot final.
- Fase 2 iniciada/consolidada: `CONTRIBUTING.md` define checklist de PR e validações; `docs/team-development.md` define fronteiras dos serviços, ownership, processo de contrato, testes e divisão de tarefas; `docs/roles.md` foi expandido com ownership de frontend, observabilidade, carga e deploy.
- Fase 3 concluída: contrato `GameService.StartMatch` (gRPC interno) + `room_id` no `PlayerInput`; Game passou a manter partidas por sala (`matches map[string]*matchState`, chave `room_id`, mais a global `__global__`); Lobby chama o Game ao iniciar (start do dono e auto-start do `SetReady`), fora do lock e com revert para `WAITING` em falha; `services/lobby` disca `GAME_GRPC_ADDR` e o Compose adiciona `depends_on: game`; cliente Phaser ganhou `lobby.ts`/`lobbyUI.ts`/`session.ts` (criar/entrar/QR/ready/start) com dep `qrcode`. Validado ponta a ponta via curl (start cria partida da sala com roster; global isolado).
- Fase 4 concluída (pipeline de tempo real): proto ganhou `PushInput(PlayerInput) returns (InputAck)` e `WatchMatch(WatchMatchRequest) returns (stream GameState)`, ambos internos (sem HTTP). Game (`internal/game/realtime.go`) roda um **relógio de servidor** por sala (`tickHz=15`) iniciado sob demanda no 1º assinante de `WatchMatch` e parado quando o último sai; inputs são bufferizados (`pendingInputs`, último vence) e consumidos por `advanceTick`; snapshots vão a `subscribers` com envio não-bloqueante (descarta em cliente lento). `StreamMatch` permanece unário/legado intacto (testes preservados). Gateway (`internal/gateway/realtime.go`, dep `github.com/gorilla/websocket`) expõe `GET /v1/match/ws?room&player`: WS→`PushInput` e `WatchMatch`→WS, com ping/pong e reescrita de `playerId`/`roomId` por autoridade. Cliente: `net.ts` ganhou `RealtimeClient` (WS, reconexão única, timeout→offline); `GameScene` virou push-based (envia input a cada `SEND_MS`, lê snapshot do WS, interpola) com fallback `OfflineDriver`; `vite.config.ts` com `ws:true`. **Decisão:** o relógio do servidor substitui o auto-restart unário da Fase 1 para partidas de sala; `StreamMatch` fica só para demo/curl. **Decisão:** snapshots WS usam `protojson` (mesmo camelCase/int64-string dos web services), então `frontend/src/types.ts` serve aos dois transportes.
- Fase 5 concluída (loop jogável): `maxMatchTicks` passou para `5*60*tickHz` (4500 ticks, ~5 min); `safeZoneAtTick` limita fase a `0..4`; `advanceTick` ignora input stale como o caminho unário; cliente envia auto-alvo ao atacar, mostra tela final com ranking (`matchEnded` + `ranking`) e o fallback offline também encerra e ranqueia a partida.

### Todos

- Fill student names and ownership roles (substituir `PLACEHOLDER` em `docs/roles.md` e `docs/report/entrega1.tex`).
- (Para submissão final) trocar o rascunho `article` pelo `sbc-template` oficial da SBC e re-verificar o limite de 4 páginas.
- Confirm Canvas dates for Entrega 1 and Entrega 2.
- Validate Hostinger VPS resources before deploy phase.
- [x] Connect `Lobby.StartRoom` to the Game service for match start. (Fase 3)
- Add request correlation/logging across services (request_id, room_id, player_id).
- Lobby start-by-time-limit (LOBB-04).
- Use the Phase 2 guide as gate for upcoming work: every task should name requirement, owner, contract impact, validation and docs affected.
- [x] **Refactor de tempo real (Fase 4):** servidor avança ticks no próprio relógio (~15 Hz) por sala + transporte WebSocket (snapshots desacoplados do request). O auto-restart unário deixou de governar partidas de sala (segue só no caminho legado `StreamMatch`).
- (Fase 7) Desconexão de cliente: hoje o último input bufferizado continua sendo aplicado se o WS cair sem enviar zerado; tratar timeout/expurgo de input por jogador inativo ao endurecer tolerância a falhas.

### Blockers

- Student roster not filled: `docs/roles.md` e o relatório usam `PLACEHOLDER`. Squads/ownership já estão definidos; faltam os nomes reais (tarefa do grupo).

## Session Continuity

Fases 1–5 implementadas e documentadas. Backend Go compila e passa nos testes
(`go test ./...`); frontend type-checka (`tsc --noEmit`). A partida em tempo
real roda por sala via WebSocket, dura até 5 minutos no relógio do servidor,
aplica baús/armas/dano/zona/ranking no Game e mostra ranking final no cliente.
O pendente administrativo segue sendo preencher os nomes reais dos alunos
(`PLACEHOLDER`).

Next recommended command:

```text
Start Phase 6 (Observability and 50-Player Stress Proof): métricas/traces + runner de 50 jogadores.
```

A Fase 6 deve medir o que a Fase 5 tornou jogável: tick rate, conexões
WebSocket, latência Gateway→Game, payload/banda, erros e simulação de 50
jogadores enviando inputs e recebendo snapshots.

---
*State updated: 2026-06-28 — Fase 5 concluída (loop jogável, safe zone de 5 minutos e ranking final; GAME-01..08). TODO aberto: preencher nomes reais (admin), observabilidade/carga na Fase 6 e tratamento de desconexão na Fase 7.*
