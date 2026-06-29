# Referência de Mensagens — Voxel Royale Distribuído

Documenta os contratos **realmente implementados**: as rotas HTTP públicas e o
WebSocket de tempo real expostos pelo Gateway, e os métodos gRPC do Lobby e do Game.

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
   │  gRPC          │  gRPC StartMatch (inicio de partida da sala)
   ▼                ▼
 Game  :50051 ◄─────┘        (estado autoritativo da partida)
```

> **Fase 3:** ao iniciar uma sala (`StartRoom` do dono ou auto-start por todos
> prontos), o Lobby chama `Game.StartMatch` via gRPC, criando uma partida
> vinculada ao `room_id`. O input do cliente passa a carregar `room_id` para
> rotear para a partida correta.
>
> **Fase 4 (tempo real):** durante a partida o navegador mantém um **WebSocket**
> com o Gateway (`/v1/match/ws`). O Gateway traduz a sessão em duas RPCs gRPC
> internas do Game: `PushInput` (cada input do cliente) e `WatchMatch` (stream de
> snapshots publicados pelo **relógio do servidor**). O Gateway faz o *fan-out*
> dos snapshots para todos os WebSockets conectados àquela sala.

```text
Navegador ──WebSocket /v1/match/ws──► Gateway :8080
   ▲                                     │  gRPC PushInput(PlayerInput)
   │                                     ▼
   │                                  Game :50051  (relógio ~15 Hz, autoritativo)
   └──────WebSocket (snapshots)◄── Gateway ◄──gRPC stream WatchMatch(GameState)
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

Transição manual `WAITING → STARTED`. **Somente o dono** pode iniciar. Ao iniciar,
o Lobby chama `Game.StartMatch` (gRPC) para criar a partida da sala com o roster
atual. Se o Game não confirmar, a sala volta para `WAITING` e o erro é `Internal`.

| Campo | Tipo | Obrigatório | Observação |
| --- | --- | --- | --- |
| `room_id` | string (path) | sim | — |
| `playerId` | string (body) | sim | Precisa ser igual a `ownerId`. |

Erros: `NotFound`, `FailedPrecondition` (sala não está em `WAITING`),
`PermissionDenied` (quem chamou não é o dono), `Internal` (Game não iniciou a partida).

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
a sala inicia automaticamente** (`WAITING → STARTED`) e o Lobby chama `Game.StartMatch`
para a sala (mesma orquestração do `StartRoom`).

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

Serviço gRPC `match.GameService` (`proto/match/v1/match.proto`). Quatro RPCs:

- `StreamMatch` — unário request/response (legado da Fase 1; demo via `curl`).
- `StartMatch` — interno, chamado pelo Lobby para criar a partida de uma sala.
- `PushInput` — interno, encaminha um input do cliente (Fase 4, NETW-03).
- `WatchMatch` — interno, *stream* de snapshots do relógio do servidor (Fase 4, NETW-04).

O servidor é autoritativo: valida o input, avança o tick e calcula o estado.

> O nome `StreamMatch` é histórico (reaproveitado da branch restaurada). Ele continua
> unário (1 tick por requisição) e serve para demonstração/compatibilidade. O fluxo de
> tempo real da Fase 4 usa `PushInput` + `WatchMatch` sob o WebSocket do Gateway.

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
| `roomId` | string | Roteia o input para a partida da sala. Vazio usa a partida global (modo demo de um jogador). |

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
  -d '{"playerId":"player-1","roomId":"room-1","moveX":1,"moveY":2,"inputSequence":1,"openChest":false,"isAttacking":false}'
```

### 3.2 `StartMatch` — gRPC interno (Lobby → Game)

Não exposto pelo Gateway (sem anotação HTTP). Criado/recriado pela transição de
sala para `STARTED`. Cria uma partida vinculada ao `room_id`, posicionando os
jogadores do roster da sala.

Request `StartMatchRequest`

| Campo | Tipo | Significado |
| --- | --- | --- |
| `roomId` | string | Sala/partida a iniciar. Obrigatório. |
| `players` | `MatchPlayer[]` | Roster (`playerId`, `playerName`) para pré-spawn. |
| `maxPlayers` | int32 | Capacidade informada pela sala. |

Response `StartMatchResponse`: `matchId` (= `roomId`) e `started` (bool).

### 3.3 `PushInput` — gRPC interno (Gateway → Game)

Encaminha um `PlayerInput` (mesmo formato do `StreamMatch`) para o **buffer** da
partida. O input não é aplicado na hora: o relógio do servidor o consome no
próximo tick. O último input por jogador vence. Resposta `InputAck` com
`accepted` (bool) e `appliedSequence` (int64, ecoa o `inputSequence`) para apoiar
a reconciliação no cliente.

### 3.4 `WatchMatch` — gRPC interno em *stream* (Game → Gateway)

`rpc WatchMatch(WatchMatchRequest) returns (stream GameState)`. O Gateway abre um
stream por WebSocket conectado; o Game envia um `GameState` por tick (relógio
~15 Hz). `WatchMatchRequest` traz `roomId` e `playerId` (o jogador aparece na
partida assim que abre o WebSocket, mesmo antes do primeiro input). O relógio da
partida é iniciado sob demanda no primeiro assinante e parado quando o último sai.

### 3.5 WebSocket público — `GET /v1/match/ws?room={roomId}&player={playerId}`

Endpoint de tempo real exposto pelo Gateway (não passa pelo grpc-gateway; é um
handler WebSocket dedicado em `internal/gateway/realtime.go`).

- **Cliente → Gateway:** mensagens de texto JSON com um `PlayerInput`
  (`playerId`/`roomId` são reescritos pelo Gateway a partir da query, por
  autoridade). Cada mensagem vira um `PushInput` gRPC.
- **Gateway → Cliente:** mensagens de texto JSON com um `GameState` por tick,
  no mesmo formato camelCase dos web services. Pings periódicos mantêm a conexão.

```bash
# Exemplo com websocat (ou o WebSocket nativo do navegador):
websocat 'ws://localhost:8080/v1/match/ws?room=room-1&player=player-1'
# enviar: {"moveX":1,"moveY":0,"inputSequence":1}
```

### Constantes de gameplay (`internal/game/server.go`)

| Parâmetro | Valor | Observação |
| --- | --- | --- |
| Arena (meia-largura) | 100 | Mundo de `-100..100` em cada eixo. |
| Movimento por tick | 2.5 | Limite do vetor de movimento. |
| Duração da partida | 4500 ticks (~5 min a 15 Hz) | Também encerra por último sobrevivente. |
| Zona segura | 90 → 8 | Encolhe linearmente; dano fora da zona: 8/tick. |
| Fases da zona | 5 | `phase` avança a cada 900 ticks e fica limitado a `0..4`. |
| Vida máxima | 100 | — |
| Armas | pistol (18 / alc. 10), rifle (24 / alc. 16), shotgun (42 / alc. 5) | dano / alcance; cooldown 1–2 ticks. |
| Baús | 9 | Posições e armas fixas (`chest-01..09`). |

---

## Itens adiados (próximas fases)

- Logs correlacionados (`request_id`, `room_id`, `player_id`) entre serviços.
- Reconexão robusta e tratamento de desconexão sem afetar a partida (Fase 7).

## Implementado na Fase 5

- Partida em tempo real passou a durar até 5 minutos no relógio do servidor
  (`4500` ticks a 15 Hz), com zona segura encolhendo até o raio final.
- Cliente Phaser exibe tela de fim de partida com ranking final vindo do
  `GameState` autoritativo (`matchEnded` + `ranking`).
- Botão/tecla de ataque usa auto-alvo no inimigo vivo mais próximo; o Game segue
  validando alcance, cooldown, dano, vida e eliminação.
- Fallback offline também encerra a partida e monta ranking local para demo.

## Implementado na Fase 4

- WebSocket `/v1/match/ws` no Gateway, mantido durante toda a partida (NETW-01).
- Cliente envia inputs sequenciados pelo WebSocket (NETW-02); o Gateway os
  encaminha ao Game via `PushInput` gRPC (NETW-03).
- Game roda um **relógio de servidor** (~15 Hz) por sala, desacoplado do ritmo
  dos clientes, e publica snapshots via `WatchMatch` que o Gateway distribui
  aos WebSockets (NETW-04). Removeu-se a dependência do auto-restart unário.
- Cliente Phaser passou a um modelo *push* (WebSocket) com interpolação dos
  jogadores remotos e fallback offline (mock) quando o backend não responde.

## Implementado na Fase 3

- `Lobby.StartRoom` e o auto-start do `SetReady` disparam `Game.StartMatch` via gRPC.
- Estado de partida por sala: o Game mantém uma partida por `room_id` (mais a
  partida global para o modo demo de um jogador).
- Cliente web com tela de lobby: criar sala, QR Code/URL, entrar por nome,
  estado pronto/aguardando e início de partida.
