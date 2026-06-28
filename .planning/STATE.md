# Project State: Voxel Royale Distribuido

## Project Reference

See: `.planning/PROJECT.md` (updated 2026-04-24)

**Core value:** Demonstrar, de forma jogavel e mensuravel, um sistema distribuido em tempo real no qual 50 jogadores participam de uma partida battle royale voxel com backend Go autoritativo e comunicacao entre servicos via gRPC.  
**Current focus:** Entrega 2 — Fase 7 concluída: tolerância a falhas, readiness, frontend no Compose e roteiro local/VPS. Próximo: Fase 8 (configurar provedor VPS e validar deploy remoto real). (Entrega 1 funcional; faltam nomes reais dos alunos.)

## Current Position

**Phase:** 7  
**Plan:** 01-01..01-05; 02-01 by-design; 03-01 QR lobby + Lobby→Game match start; 04-01 realtime pipeline; 05-01 playable loop + final ranking; 06-01 observability + stress proof; 07-01 fault tolerance + deploy readiness  
**Status:** Distributed skeleton + by-design docs + Fases 3, 4, 5, 6 e 7 completas. Game roda relógio de servidor (15 Hz) por sala com `PushInput`/`WatchMatch`, duração de até 4500 ticks (~5 min), safe zone progressiva, baús/armas/dano/eliminação/ranking; Gateway expõe WebSocket `/v1/match/ws`; cliente Phaser usa modelo push (`RealtimeClient`). Observabilidade: `/metrics` Prometheus nos três serviços, OpenTelemetry HTTP/gRPC exportando OTLP para Jaeger no Compose, Grafana provisionado e runner `tools/stress50`/`make stress50`. Tolerância/deploy readiness: Game limpa input pendente ao desconectar jogador, Gateway/Lobby/Game expõem `/readyz`, Lobby trata falha Lobby→Game com timeout/revert, Compose sobe frontend Nginx + backends + telemetria, e `docs/deploy.md` documenta local/VPS. Deploy remoto real ainda não foi validado e agora é a Fase 8 (`DEPL-04`). Faltam nomes reais dos alunos.  
**Progress:** █████████░ ~87%

## Performance Metrics

- Requirements total: 41
- Requirements mapped: 41
- Phases total: 9
- Phase 1 plans complete: 5/5 (01-05 done; report draft com nomes a preencher)
- Phase 2 plans complete: 1/1 (`ARCH-06`)
- Phase 3 plans complete: 1/1 (`LOBB-01..03` done, `LOBB-04` partial)
- Phase 4 plans complete: 1/1 (`NETW-01..04` done)
- Phase 5 plans complete: 1/1 (`GAME-01..08` done)
- Phase 6 plans complete: 1/1 (`NETW-05`, `OBSV-01..05` done)
- Phase 7 plans complete: 1/1 (`FAIL-01..03`, `DEPL-01..03` done)
- Phase 8 plans complete: 0/0 (`DEPL-04` pending: provider VPS real)
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
- Fase 6 concluída (observabilidade + carga): pacote `internal/observability` centraliza Prometheus e OpenTelemetry; Gateway/Lobby/Game expõem `/metrics`; Gateway mede WebSocket, bytes, erros e latência `PushInput`; Game mede inputs, streams, ticks, duração do tick, assinantes, jogadores e drops; Lobby mede salas/jogadores/eventos. Compose adicionou Prometheus (`:9090`), Grafana (`:3000`) com dashboard provisionado e Jaeger (`:16686`) recebendo OTLP HTTP (`:4318`). `tools/stress50`/`make stress50` cria sala, conecta bots via WebSocket e envia inputs; `docs/stress-results.md` registrou smoke local com 50/50 conexões, 3.750 inputs/snapshots em 5s e zero erros.
- Fase 7 concluída (falhas + deploy readiness): `subscriber` agora carrega `playerID`; `matchState.connectedPlayers` conta conexões por jogador; ao cair o último `WatchMatch`, o Game remove o input pendente do jogador e impede movimento/ataque fantasma. Gateway/Lobby/Game expõem `/readyz`; Gateway checa TCP para Game/Lobby; Lobby checa Game e limita `StartMatch` a 3s; falha de start continua revertendo sala para `WAITING`. Compose inclui `frontend` (Vite build + Nginx proxy para Gateway/WebSocket) e usa readiness nos healthchecks. `docs/deploy.md` documenta local/VPS, portas, readiness e degradação controlada. **Importante:** a validação em VPS real não foi executada; foi separada para a Fase 8.
- Fase 8 criada (VPS provider setup + remote deploy): requisito novo `DEPL-04` cobre escolher/configurar provedor (Hostinger ou equivalente), confirmar recursos, configurar SSH/firewall/Docker Compose, subir stack remota e capturar prova publica (`frontend-healthz`, `/readyz`, `/metrics`, stress remoto).

### Todos

- Fill student names and ownership roles (substituir `PLACEHOLDER` em `docs/roles.md` e `docs/report/entrega1.tex`).
- (Para submissão final) trocar o rascunho `article` pelo `sbc-template` oficial da SBC e re-verificar o limite de 4 páginas.
- Confirm Canvas dates for Entrega 1 and Entrega 2.
- Validate Hostinger/VPS provider resources and credentials before Phase 8 remote deploy.
- [x] Connect `Lobby.StartRoom` to the Game service for match start. (Fase 3)
- Add request correlation/logging across services (request_id, room_id, player_id).
- Lobby start-by-time-limit (LOBB-04).
- Use the Phase 2 guide as gate for upcoming work: every task should name requirement, owner, contract impact, validation and docs affected.
- [x] **Refactor de tempo real (Fase 4):** servidor avança ticks no próprio relógio (~15 Hz) por sala + transporte WebSocket (snapshots desacoplados do request). O auto-restart unário deixou de governar partidas de sala (segue só no caminho legado `StreamMatch`).
- [x] **Observabilidade/carga (Fase 6):** Prometheus + OpenTelemetry + Grafana/Jaeger + runner `stress50` e resultado de 50 jogadores registrado.
- [x] **Tolerância/deploy (Fase 7):** desconexão limpa input pendente; `/readyz`; frontend no Compose; guia local/VPS.

### Blockers

- Student roster not filled: `docs/roles.md` e o relatório usam `PLACEHOLDER`. Squads/ownership já estão definidos; faltam os nomes reais (tarefa do grupo).

## Session Continuity

Fases 1–7 implementadas e documentadas. Backend Go compila e passa nos testes
(`go test ./...`); frontend builda (`npm run build`); Compose completo subiu com
frontend, Gateway, Lobby, Game, Prometheus, Grafana e Jaeger; `frontend-healthz`,
Gateway `/readyz` e `/metrics` responderam OK. Smoke local `tools/stress50`
conectou 50/50 bots e recebeu 3.750 snapshots em 5s. O deploy remoto real ainda
depende de provedor/acesso VPS e agora é a Fase 8. O pendente administrativo segue
sendo preencher os nomes reais dos alunos (`PLACEHOLDER`).

Next recommended command:

```text
Start Phase 8 (VPS Provider Setup and Remote Deploy): escolher/configurar VPS, SSH/firewall/Docker, subir stack remota e validar endpoints publicos.
```

A Fase 8 deve transformar o roteiro local em deploy remoto real: selecionar Hostinger
ou provedor equivalente, confirmar recursos, configurar acesso SSH/firewall/Docker,
subir o Compose e registrar evidências dos endpoints públicos e do stress remoto.

---
*State updated: 2026-06-28 — Fase 8 criada para configuração real de provedor VPS e deploy remoto (`DEPL-04`). Fase 7 segue concluída como readiness local/deploy guide; TODO aberto: obter acesso/provedor VPS, validar remoto, preencher nomes reais e depois preparar relatório/apresentação na Fase 9.*
