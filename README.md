# Voxel Royale

Monorepo do Voxel Royale, um jogo distribuido com cliente web, Gateway HTTP/WebSocket,
servicos gRPC e observabilidade. A execucao completa usa Docker Compose e sobe
frontend, Gateway, Game, Lobby primario, Lobby backup, Prometheus, Grafana e Jaeger.

- `frontend/`: cliente web em Phaser 3 + TypeScript + Vite.
- `services/gateway`: entrada HTTP/WebSocket publica e traducao para gRPC.
- `services/game`: servico gRPC autoritativo para movimento, baus, armas, dano,
  safe zone, ranking e partidas por sala (`room_id`).
- `services/lobby`: gestao de salas (criar/entrar/pronto/iniciar), integracao
  com o Game via `StartMatch` e suporte a replicacao primario-backup.
- `deployments/`: Docker Compose, Prometheus e Grafana.

## Estrutura

```text
.
├── deployments/docker-compose.yml
├── frontend/               # cliente Phaser/Vite
├── gen/                    # codigo Go gerado a partir dos contratos
├── internal/
│   ├── gateway/
│   ├── game/
│   ├── lobby/
│   └── observability/
├── proto/
│   ├── lobby/v1/lobby.proto
│   └── match/v1/match.proto
├── services/
│   ├── gateway/
│   ├── game/
│   └── lobby/
└── tools/stress50/         # runner de carga com jogadores simultaneos
```

## Configuracao do Ambiente

### Para executar o sistema completo

- Docker Desktop com Docker Compose v2.
- No Windows, WSL2 habilitado para o Docker Desktop.

### Para desenvolvimento local

- Go `1.25.0` ou superior.
- Node.js e npm para o cliente web.

### Para regenerar os contratos protobuf

Estes passos so sao necessarios quando os arquivos `.proto` forem alterados. O
codigo gerado em `gen/` ja esta versionado para executar o projeto normalmente.

Instale o `protoc`:

- Baixe o release mais recente do Protocol Buffers para Windows:
  [github.com/protocolbuffers/protobuf/releases](https://github.com/protocolbuffers/protobuf/releases).
- Descompacte o ZIP.
- Coloque `protoc.exe` em uma pasta no `PATH`, como `C:\Windows\System32`, ou
  adicione uma pasta propria ao `PATH`.

Baixe as dependencias de protos do Google:

```bash
mkdir third_party
cd third_party
git clone --depth 1 https://github.com/googleapis/googleapis
```

Instale os geradores Go:

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
```

Regere os arquivos com:

```bash
make proto
```

## Como Executar

### Execucao completa com Docker

Este e o caminho recomendado para demonstrar o projeto completo, incluindo
frontend, backend, replicacao do Lobby, failover e observabilidade.

```bash
docker compose -f deployments/docker-compose.yml config
docker compose -f deployments/docker-compose.yml up --build
```

Atalhos equivalentes pelo Makefile:

```bash
make compose-config
make docker-up
```

Endpoints principais:

- Frontend: `http://localhost:5173`
- Gateway HTTP/WebSocket: `http://localhost:8080`
- Health do Gateway: `curl http://localhost:8080/healthz`
- Readiness do Gateway: `curl http://localhost:8080/readyz`
- Health do frontend: `curl http://localhost:5173/frontend-healthz`
- Prometheus: `http://localhost:9090`
- Grafana: `http://localhost:3000/d/voxel-royale/voxel-royale`
- Jaeger: `http://localhost:16686/search`

Para parar a stack:

```bash
docker compose -f deployments/docker-compose.yml down
```

### Execucao local para desenvolvimento

Este modo e util para depurar servicos diretamente com `go run`, mas sobe apenas
um Lobby standalone. Ele nao ativa a topologia primario-backup nem o failover; use
Docker Compose para validar o comportamento completo.

Em terminais separados, a partir da raiz do repositorio:

```bash
go run ./services/game
go run ./services/lobby
go run ./services/gateway
```

Para rodar o cliente web em modo dev:

```bash
cd frontend
npm install
npm run dev      # http://localhost:5173
```

O Vite encaminha chamadas `/v1`, `/healthz` e `/readyz` para o Gateway em
`http://localhost:8080`.

## Testando a API

O Gateway recebe requisicoes HTTP na porta `8080` e conversa com os servicos
internos por gRPC.

Health check:

```bash
curl http://localhost:8080/healthz
```

Readiness:

```bash
curl http://localhost:8080/readyz
```

Criar sala:

```bash
curl -X POST http://localhost:8080/v1/rooms \
  -H "Content-Type: application/json" \
  -d "{\"ownerName\":\"Player 1\",\"maxPlayers\":4}"
```

Enviar movimento pela rota HTTP de compatibilidade:

```bash
curl -X POST http://localhost:8080/v1/match/stream \
  -H "Content-Type: application/json" \
  -d "{\"playerId\":\"player-1\",\"moveX\":1,\"moveY\":2,\"inputSequence\":1,\"openChest\":false,\"isAttacking\":false}"
```

O jogo em tempo real usa WebSocket em `GET /v1/match/ws`, com o Gateway
traduzindo as mensagens para `PushInput` e `WatchMatch` no Game.

## Frontend / Cliente

O cliente web em [`frontend/`](frontend/) usa Phaser 3 + TypeScript + Vite e
renderiza uma arena top-down ".io" com grama, rio, arvores, pedras, jogador
controlavel, zona segura, baus e demais jogadores a partir do `GameState`
autoritativo.

- Ao abrir, o cliente mostra a tela de sala: criar sala com QR Code/URL, entrar
  por nome ou via `?room=<id>`, marcar pronto e iniciar a partida.
- Em modo ao vivo, mantem WebSocket com o Gateway (`GET /v1/match/ws`) e recebe
  snapshots do relogio do servidor.
- Se o Gateway nao responder, cai em um simulador local para continuar jogavel.
- Controles: WASD/setas ou joystick para mover; espaco/ATIRAR para atacar; E/BAU
  para abrir bau.

## Observabilidade e Carga

Com a stack Docker ativa, Prometheus, Grafana e Jaeger sobem junto com os
servicos:

```bash
docker compose -f deployments/docker-compose.yml up --build
make stress50
```

- Metricas Prometheus: `http://localhost:8080/metrics`, `:8081/metrics`,
  `:8082/metrics`.
- Dashboard Grafana: `http://localhost:3000/d/voxel-royale/voxel-royale`.
- Traces Jaeger: `http://localhost:16686/search`.
- Runner de carga: `go run ./tools/stress50 -players 50 -duration 30s`.

## Tolerancia a Falhas e Deploy Local

O Compose sobe o sistema completo: frontend, Gateway, Lobby primario, Lobby
backup, Game, Prometheus, Grafana e Jaeger.

O Gateway promove automaticamente o Lobby backup quando o primario fica
indisponivel, mantendo o fluxo de criacao e consulta de salas ativo apos o
failover.

Validacoes uteis:

```bash
curl http://localhost:5173/frontend-healthz
curl http://localhost:8080/readyz
curl http://localhost:8080/metrics
```

Para testar failover do Lobby:

```bash
docker compose -f deployments/docker-compose.yml stop lobby-primary
curl -X POST http://localhost:8080/v1/rooms \
  -H "Content-Type: application/json" \
  -d "{\"ownerName\":\"Failover\",\"maxPlayers\":4}"
curl http://localhost:8080/readyz
```

O Gateway chama `POST /replication/promote` no `lobby-backup`, repete a operacao
no backup promovido e passa a encaminhar as proximas operacoes de sala para ele.
A metrica `voxel_gateway_lobby_failovers_total` registra a troca.

## Smoke Test

Com a stack ativa:

```powershell
curl http://localhost:8080/healthz
curl -X POST http://localhost:8080/v1/match/stream `
  -H "Content-Type: application/json" `
  -d "{\"playerId\":\"player-1\",\"moveX\":1,\"moveY\":2,\"inputSequence\":1,\"openChest\":false,\"isAttacking\":false}"
```

Trecho esperado do fluxo HTTP Gateway -> gRPC Game:

```json
{
  "tick": "1",
  "players": [
    {
      "playerId": "player-1",
      "x": 1,
      "y": 2,
      "isAlive": true,
      "health": 100,
      "weapon": "pistol"
    }
  ],
  "chests": [
    {
      "chestId": "chest-01",
      "x": 3,
      "y": 0,
      "weapon": "rifle"
    }
  ],
  "safeZone": {
    "centerX": 0,
    "centerY": 0,
    "radius": 44.866665,
    "phase": "0"
  },
  "ranking": [
    {
      "playerId": "player-1",
      "place": 1,
      "isAlive": true,
      "health": 100
    }
  ],
  "remainingTicks": "299"
}
```

## Notas Tecnicas

- A raiz do repositorio e o monorepo Go `voxel-royale`.
- O codigo gerado fica em `gen/`.
- Cada servico tem Dockerfile proprio e healthcheck.
- Em container, o Gateway usa DNS interno do Compose (`game:50051`,
  `lobby-primary:50052` e `lobby-backup:50052`) em vez de `localhost`.
- O Lobby e separado e containerizado, com salas completas, integracao
  Lobby->Game, replicacao primario-backup e promocao do backup.
- O Game mantem estado em memoria para movimentacao validada, abertura de baus,
  armas, dano, eliminacao, safe zone e ranking.
