# Project State: Voxel Royale Distribuido

## Project Reference

See: `.planning/PROJECT.md` (updated 2026-04-24)

**Core value:** Demonstrar, de forma jogavel e mensuravel, um sistema distribuido em tempo real no qual 50 jogadores participam de uma partida battle royale voxel com backend Go autoritativo e comunicacao entre servicos via gRPC.  
**Current focus:** Entrega 2 — Fase 4 concluída: pipeline de tempo real (WebSocket no Gateway + relógio do servidor no Game; inputs e snapshots em tempo real). Próximo: Fase 5 (loop de partida jogável + tela de fim/ranking). (Entrega 1 funcional; faltam nomes reais dos alunos.)

## Current Position

**Phase:** 4  
**Plan:** 01-01..01-05; 02-01 by-design; 03-01 QR lobby + Lobby→Game match start; 04-01 realtime pipeline  
**Status:** Distributed skeleton + by-design docs + Fase 3 + Fase 4 completas. Game roda relógio de servidor (~15 Hz) por sala com `PushInput`/`WatchMatch`; Gateway expõe WebSocket `/v1/match/ws` ligando WS↔gRPC com fan-out de snapshots; cliente Phaser usa modelo push (`RealtimeClient`) com interpolação e fallback offline. Validado ponta a ponta (WS → 22 snapshots/1.5s, tick avança no relógio, input encaminhado). Faltam nomes reais dos alunos.  
**Progress:** █████░░░░░ ~50%

## Performance Metrics

- Requirements total: 40
- Requirements mapped: 40
- Phases total: 8
- Phase 1 plans complete: 5/5 (01-05 done; report draft com nomes a preencher)
- Phase 2 plans complete: 1/1 (`ARCH-06`)
- Phase 3 plans complete: 1/1 (`LOBB-01..03` done, `LOBB-04` partial)
- Phase 4 plans complete: 1/1 (`NETW-01..04` done)
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
- Fase 4 concluída (pipeline de tempo real): proto ganhou `PushInput(PlayerInput) returns (InputAck)` e `WatchMatch(WatchMatchRequest) returns (stream GameState)`, ambos internos (sem HTTP). Game (`internal/game/realtime.go`) roda um **relógio de servidor** por sala (`tickHz=15`) iniciado sob demanda no 1º assinante de `WatchMatch` e parado quando o último sai; inputs são bufferizados (`pendingInputs`, último vence) e consumidos por `advanceTick`; snapshots vão a `subscribers` com envio não-bloqueante (descarta em cliente lento). `StreamMatch` permanece unário/legado intacto (testes preservados). Gateway (`internal/gateway/realtime.go`, dep `github.com/gorilla/websocket`) expõe `GET /v1/match/ws?room&player`: WS→`PushInput` e `WatchMatch`→WS, com ping/pong e reescrita de `playerId`/`roomId` por autoridade. Cliente: `net.ts` ganhou `RealtimeClient` (WS, reconexão única, timeout→offline); `GameScene` virou push-based (envia input a cada `SEND_MS`, lê snapshot do WS, interpola) com fallback `OfflineDriver`; `vite.config.ts` com `ws:true`. **Decisão:** o relógio do servidor substitui o auto-restart unário da Fase 1 para partidas de sala; `StreamMatch` fica só para demo/curl. **Decisão:** snapshots WS usam `protojson` (mesmo camelCase/int64-string dos web services), então `frontend/src/types.ts` serve aos dois transportes.

### Todos

- Fill student names and ownership roles (substituir `PLACEHOLDER` em `docs/roles.md` e `docs/report/entrega1.tex`).
- (Para submissão final) trocar o rascunho `article` pelo `sbc-template` oficial da SBC e re-verificar o limite de 4 páginas.
- Confirm Canvas dates for Entrega 1 and Entrega 2.
- Validate Hostinger VPS resources before deploy phase.
- [x] Connect `Lobby.StartRoom` to the Game service for match start. (Fase 3)
- Add request correlation/logging across services (request_id, room_id, player_id).
- Lobby start-by-time-limit (LOBB-04) e tela de fim de partida/ranking no cliente (Fase 5).
- Use the Phase 2 guide as gate for upcoming work: every task should name requirement, owner, contract impact, validation and docs affected.
- [x] **Refactor de tempo real (Fase 4):** servidor avança ticks no próprio relógio (~15 Hz) por sala + transporte WebSocket (snapshots desacoplados do request). O auto-restart unário deixou de governar partidas de sala (segue só no caminho legado `StreamMatch`).
- (Fase 7) Desconexão de cliente: hoje o último input bufferizado continua sendo aplicado se o WS cair sem enviar zerado; tratar timeout/expurgo de input por jogador inativo ao endurecer tolerância a falhas.

### Blockers

- Student roster not filled: `docs/roles.md` e o relatório usam `PLACEHOLDER`. Squads/ownership já estão definidos; faltam os nomes reais (tarefa do grupo).

## Session Continuity

Fases 1–4 implementadas e documentadas. Backend Go compila e passa nos testes
(`go test ./...`); frontend type-checka (`tsc --noEmit`). Pipeline de tempo real
validado ponta a ponta (Game+Gateway locais, cliente WebSocket Node): WS abre,
relógio do servidor avança o tick, input chega ao Game via gRPC e snapshots
voltam ao cliente. O pendente administrativo segue sendo preencher os nomes
reais dos alunos (`PLACEHOLDER`).

Next recommended command:

```text
Start Phase 5 (Playable Voxel Battle Royale): loop completo de partida + controles touch + tela de fim/ranking.
```

A Fase 5 se apoia no pipeline da Fase 4: o cliente já recebe snapshots em tempo
real (jogadores, baús, armas, zona, ranking) — falta amarrar o loop jogável
(spawns/itens/eliminação afinados) e a tela de fim de partida com ranking.

---
*State updated: 2026-06-28 — Fase 4 concluída (pipeline de tempo real: WebSocket no Gateway + relógio do servidor no Game; NETW-01..04). TODO aberto: preencher nomes reais (admin) e tratamento de desconexão na Fase 7.*
