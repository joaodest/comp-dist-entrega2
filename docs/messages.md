# Referência de Mensagens — Voxel Royale Distribuído

Documenta os contratos **realmente implementados** na Fase 1 (Entrega 1): as rotas
HTTP públicas expostas pelo Gateway e os métodos gRPC do Lobby e do Game.

Fontes de verdade:

- `proto/lobby/v1/lobby.proto` (serviço `LobbyService`)
- `proto/match/v1/match.proto` (serviço `GameService`)
- `internal/gateway/handler.go` (montagem do proxy HTTP→gRPC)
- `deployments/docker-compose.yml` (portas e DNS de serviço)

> **Tradução HTTP↔gRPC:** o Gateway usa [grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway).
> As anotações `google.api.http` em cada `.proto` definem a rota HTTP correspondente a cada RPC.
> Na serialização JSON do grpc-gateway os campos saem em **camelCase** (`player_id` → `playerId`,
> `room_id` → `roomId`, `move_x` → `moveX`), seguindo o mapeamento JSON padrão do proto3.

---

## Topologia de portas

| Serviço | Papel | Interface | Porta interna | Exposição |
| --- | --- | --- | --- | --- |
| Gateway | Borda HTTP pública | HTTP (REST/JSON) | `:8080` | publicada no host (`8080:8080`) |
| Game | Backend autoritativo de partida | gRPC `GameService` | `:50051` | só na rede do Compose (`game:50051`) |
| Game | Health check | HTTP | `:8082` | só na rede do Compose |
| Lobby | Gestor de salas | gRPC `LobbyService` | `:50052` | só na rede do Compose (`lobby:50052`) |
| Lobby | Health check | HTTP | `:8081` | só na rede do Compose |

O Gateway descobre os back-ends por variáveis de ambiente
(`GAME_GRPC_ADDR=game:50051`, `LOBBY_GRPC_ADDR=lobby:50052`).

```text
Navegador / curl
      │  HTTP JSON
      ▼
  Gateway  :8080
   │            │  gRPC (protobuf)
   │            ▼
   │        Lobby  :50052   (ciclo de vida da sala)
   │  gRPC (protobuf)
   ▼
 Game  :50051               (estado autoritativo da partida)
```

---

## 1. Gateway — rotas utilitárias

### `GET /healthz`

Liveness do Gateway. Não toca em gRPC.

```text
$ curl http://localhost:8080/healthz
ok
```

Cada back-end também expõe seu próprio `/healthz` (Game em `:8082`, Lobby em `:8081`),
usados pelos `healthcheck` do Docker Compose.

---

## 2. LobbyService — ciclo de vida das salas

Serviço gRPC `lobby.LobbyService` (`proto/lobby/v1/lobby.proto`). Todas as respostas
são um `RoomResponse`.

### Tipos compartilhados

`RoomResponse`

| Campo | Tipo | Significado |
| --- | --- | --- |
| `roomId` | string | Identificador da sala (`room-{n}`). |
| `status` | enum `RoomStatus` | `ROOM_STATUS_WAITING`, `ROOM_STATUS_STARTED` ou `ROOM_STATUS_CLOSED`. |
| `ownerId` | string | `playerId` do dono atual da sala. |
| `players` | `Player[]` | Jogadores presentes. |
| `maxPlayers` | int32 | Capacidade da sala (padrão 50). |
| `joinUrl` | string | Caminho relativo para entrada (`/v1/rooms/{roomId}/join`). |

`Player`

| Campo | Tipo | Significado |
| --- | --- | --- |
| `playerId` | string | ID atribuído pelo Lobby. Dono: `player-{n}-1`; demais: `player-{roomId}-{seq}`. |
| `playerName` | string | Nome de exibição informado pelo jogador. |
| `ready` | bool | Se o jogador marcou "pronto". |

### 2.1 `CreateRoom` — `POST /v1/rooms`

Cria uma sala em estado `WAITING`. Quem cria vira o dono e já entra como primeiro jogador.

Request

| Campo | Tipo | Obrigatório | Observação |
| --- | --- | --- | --- |
| `ownerName` | string | sim | Rejeita vazio (`InvalidArgument`). |
| `maxPlayers` | int32 | não | `<= 0` usa o padrão 50. |

```bash
curl -X POST http://localhost:8080/v1/rooms \
  -H 'Content-Type: application/json' \
  -d '{"ownerName":"Ana","maxPlayers":10}'
```

### 2.2 `JoinRoom` — `POST /v1/rooms/{room_id}/join`

Adiciona um jogador a uma sala existente em `WAITING`.

| Campo | Tipo | Obrigatório | Observação |
| --- | --- | --- | --- |
| `room_id` | string (path) | sim | — |
| `playerName` | string (body) | sim | — |

Erros: `NotFound` (sala inexistente), `FailedPrecondition` (sala não está em `WAITING` ou está cheia).

```bash
curl -X POST http://localhost:8080/v1/rooms/room-1/join \
  -H 'Content-Type: application/json' \
  -d '{"playerName":"Bruno"}'
```

### 2.3 `GetRoom` — `GET /v1/rooms/{room_id}`

Retorna o estado atual da sala.

```bash
curl http://localhost:8080/v1/rooms/room-1
```

Erro: `NotFound` se a sala não existir.

### 2.4 `StartRoom` — `POST /v1/rooms/{room_id}/start`

Transição manual `WAITING → STARTED`. **Somente o dono** pode iniciar.

| Campo | Tipo | Obrigatório | Observação |
| --- | --- | --- | --- |
| `room_id` | string (path) | sim | — |
| `playerId` | string (body) | sim | Precisa ser igual a `ownerId`. |

Erros: `NotFound`, `FailedPrecondition` (sala não está em `WAITING`),
`PermissionDenied` (quem chamou não é o dono).

```bash
curl -X POST http://localhost:8080/v1/rooms/room-1/start \
  -H 'Content-Type: application/json' \
  -d '{"playerId":"player-1-1"}'
```

### 2.5 `LeaveRoom` — `POST /v1/rooms/{room_id}/leave`

Remove um jogador da sala.

- Se o dono sai e ainda há jogadores, a posse é transferida ao primeiro da lista.
- Se a sala fica vazia, ela é fechada (`ROOM_STATUS_CLOSED`) e removida.

Erros: `NotFound` (sala ou jogador inexistente).

### 2.6 `SetReady` — `POST /v1/rooms/{room_id}/ready`

Marca/desmarca um jogador como pronto. **Quando todos os jogadores presentes estão prontos,
a sala inicia automaticamente** (`WAITING → STARTED`).

| Campo | Tipo | Obrigatório | Observação |
| --- | --- | --- | --- |
| `room_id` | string (path) | sim | — |
| `playerId` | string (body) | sim | Precisa pertencer à sala. |
| `ready` | bool (body) | sim | `true` = pronto, `false` = desfaz. |

Erros: `NotFound`, `FailedPrecondition` (sala não está em `WAITING`).

```bash
curl -X POST http://localhost:8080/v1/rooms/room-1/ready \
  -H 'Content-Type: application/json' \
  -d '{"playerId":"player-room-1-2","ready":true}'
```

---

## 3. GameService — partida autoritativa

Serviço gRPC `match.GameService` (`proto/match/v1/match.proto`). Hoje há **um único RPC**:
`StreamMatch`, que recebe um input e devolve o snapshot completo da partida. O servidor é
autoritativo: ele valida o input, avança um tick e calcula o estado.

> O nome `StreamMatch` é histórico (reaproveitado da branch restaurada). Apesar do nome, no
> momento é uma chamada unária request/response, não um stream gRPC. O início de partida
> orquestrado por `Lobby.StartRoom → Game` e o transporte WebSocket ficam para fases futuras.

### 3.1 `StreamMatch` — `POST /v1/match/stream`

Request `PlayerInput`

| Campo | Tipo | Significado |
| --- | --- | --- |
| `playerId` | string | Obrigatório. O jogador é criado no primeiro input (spawn). |
| `moveX`, `moveY` | float | Vetor de movimento desejado; é normalizado/limitado a `maxMovePerTick = 2.5`. |
| `isAttacking` | bool | Dispara ataque com a arma atual (respeita cooldown). |
| `inputSequence` | int64 | Sequência monotônica; inputs antigos/repetidos são descartados. |
| `openChest` | bool | Abre o baú mais próximo dentro do alcance (`2.25`). |
| `targetPlayerId` | string | Alvo explícito de ataque (opcional). |
| `aimX`, `aimY` | float | Direção de mira para seleção automática de alvo. |

Response `GameState`

| Campo | Tipo | Significado |
| --- | --- | --- |
| `tick` | int64 | Tick atual da partida (incrementa a cada input processado). |
| `players` | `PlayerSnapshot[]` | Posição, vida, arma, eliminações, dano e ticks sobrevividos. |
| `chests` | `ChestSnapshot[]` | Baús, posição, se foram abertos e por quem. |
| `safeZone` | `SafeZoneSnapshot` | Centro, raio e fase da zona segura. |
| `ranking` | `RankingEntry[]` | Classificação ordenada (vivos primeiro, depois por eliminações/sobrevivência). |
| `matchEnded` | bool | `true` quando resta ≤1 jogador vivo ou o limite de ticks é atingido. |
| `remainingTicks` | int64 | Ticks restantes até o fim da partida. |

```bash
curl -X POST http://localhost:8080/v1/match/stream \
  -H 'Content-Type: application/json' \
  -d '{"playerId":"player-1","moveX":1,"moveY":2,"inputSequence":1,"openChest":false,"isAttacking":false}'
```

### Constantes de gameplay (`internal/game/server.go`)

| Parâmetro | Valor | Observação |
| --- | --- | --- |
| Arena (meia-largura) | 50 | Mundo de `-50..50` em cada eixo. |
| Movimento por tick | 2.5 | Limite do vetor de movimento. |
| Duração da partida | 300 ticks | Também encerra por último sobrevivente. |
| Zona segura | 45 → 5 | Encolhe linearmente; dano fora da zona: 8/tick. |
| Fases da zona | 5 | `phase = tick / 60`. |
| Vida máxima | 100 | — |
| Armas | pistol (18 / alc. 10), rifle (24 / alc. 16), shotgun (42 / alc. 5) | dano / alcance; cooldown 1–2 ticks. |
| Baús | 9 | Posições e armas fixas (`chest-01..09`). |

---

## Itens adiados (fora da Fase 1)

- `Lobby.StartRoom` ainda **não** dispara a partida no Game via gRPC.
- Transporte WebSocket de inputs/snapshots em tempo real.
- Logs correlacionados (`request_id`, `room_id`, `player_id`) entre serviços.
- Estado de partida por sala (hoje o Game mantém uma única partida em memória).
