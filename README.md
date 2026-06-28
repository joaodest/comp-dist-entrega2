# Voxel Royale

Monorepo do backend distribuido do Voxel Royale. A estrutura atual separa os tres servicos planejados para a Entrega 1:

- `services/gateway`: entrada HTTP publica e traducao HTTP -> gRPC para Game e Lobby.
- `services/game`: servico gRPC autoritativo para movimento, baus, armas, dano, safe zone e ranking; mantem partidas por sala (`room_id`).
- `services/lobby`: gestao de salas (criar/entrar/pronto/iniciar) que dispara a partida no Game via gRPC (`StartMatch`).

## Estrutura

```text
.
├── deployments/docker-compose.yml
├── gen/                    # codigo Go gerado a partir dos contratos
├── internal/
│   ├── gateway/
│   ├── game/
│   └── lobby/
├── proto/
│   ├── lobby/v1/lobby.proto
│   └── match/v1/match.proto
└── services/
    ├── gateway/
    ├── game/
    └── lobby/
```

## Configuração do Ambiente

### Pré-requisitos

#### Instalação do Docker e WSL2
- Baixe e instale o Docker Desktop para Windows: [docker.com](https://www.docker.com/products/docker-desktop/)
- Certifique-se de que o WSL2 está habilitado (Docker Desktop instala automaticamente, mas verifique com `wsl --list --verbose`)

#### Instalação do Go (Golang)
- Baixe a versão 1.25.0 ou superior do Go: [golang.org](https://golang.org/dl/)
- Instale e adicione ao PATH do sistema
- Verifique com `go version`

#### Download do Compilador Protoc (Protocol Buffers)
- Baixe o release mais recente do protoc para Windows: [github.com/protocolbuffers/protobuf/releases](https://github.com/protocolbuffers/protobuf/releases)
- Descompacte o arquivo ZIP
- Mova o executável `protoc.exe` da pasta `bin` para uma pasta no PATH do sistema (ex: `C:\Windows\System32` ou adicione uma pasta personalizada ao PATH)

### Dependências de Terceiros (Protos)

Crie a pasta `third_party` e clone o repositório de APIs do Google:

```bash
mkdir third_party
cd third_party
git clone --depth 1 https://github.com/googleapis/googleapis
```

### Plugins do Go para Protobuf

Instale os geradores de código via terminal:

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
```

## Como Executar

### Executando o Projeto com Docker

Valide a configuração e suba os containers:

```bash
docker compose -f deployments/docker-compose.yml config
docker compose -f deployments/docker-compose.yml up --build
```

### Testando a API

O Gateway recebe requisições HTTP na porta 8080 e converte para gRPC internamente.

#### Health Check
```bash
curl http://localhost:8080/healthz
```

#### Enviar Movimento (Stream)
```bash
curl -X POST http://localhost:8080/v1/match/stream \
  -H "Content-Type: application/json" \
  -d "{\"playerId\":\"player-1\",\"moveX\":1,\"moveY\":2,\"inputSequence\":1,\"openChest\":false,\"isAttacking\":false}"
```

## Documentacao

- [Architecture update](docs/architecture.md): arquitetura implementada em relacao ao plano original.
- [Implementation delta](docs/implementation-delta.md): checklist do que foi feito, desvios e gaps restantes.
- [Team development](docs/team-development.md): guia by-design da Fase 2 para ownership, contratos, testes e reviews.
- [Observability and stress](docs/observability.md): Prometheus/Grafana/Jaeger e runner de 50 jogadores da Fase 6.
- [Stress results](docs/stress-results.md): smoke local com 50 jogadores simultâneos.
- [Contributing](CONTRIBUTING.md): checklist curto para contribuir e validar PRs.

## Frontend / Cliente (Phaser 2D)

Cliente web do jogo em [`frontend/`](frontend/) (Phaser 3 + TypeScript + Vite), estilo
top-down ".io" (grama, rio, árvores, pedras, jogador controlável, zona segura e baús
renderizados a partir do `GameState`).

```bash
cd frontend
npm install
npm run dev      # http://localhost:5173
```

- **Lobby (Fase 3):** ao abrir, o cliente mostra a tela de sala — criar sala (com QR Code/URL),
  entrar por nome (inclusive via `?room=<id>`), marcar pronto e iniciar a partida. Para o fluxo
  completo de salas suba os três serviços: `go run ./services/game`, `go run ./services/lobby` e
  `go run ./services/gateway`.
- **AO VIVO:** durante a partida mantém WebSocket com o Gateway (`GET /v1/match/ws`, via proxy do Vite para `:8080`),
  enviando inputs sequenciados e recebendo snapshots do relógio do servidor.
- **OFFLINE (mock):** se o Gateway não responder, cai num simulador local e continua jogável.
- Controles: WASD/setas ou joystick para mover; espaço/ATIRAR; E/BAÚ. Detalhes em
  [`frontend/README.md`](frontend/README.md).

> `POST /v1/match/stream` permanece como demo/compatibilidade por `curl`; o jogo real usa
> WebSocket + `PushInput`/`WatchMatch`.

## Observabilidade e Carga (Fase 6)

Com a stack Docker ativa, Prometheus, Grafana e Jaeger tambem sobem para demonstrar
o comportamento distribuido:

```bash
docker compose -f deployments/docker-compose.yml up --build
make stress50
```

- Métricas Prometheus: `http://localhost:8080/metrics`, `:8081/metrics`, `:8082/metrics`.
- Dashboard Grafana: `http://localhost:3000/d/voxel-royale/voxel-royale`.
- Traces Jaeger: `http://localhost:16686/search`.
- Runner de carga: `go run ./tools/stress50 -players 50 -duration 30s`.

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

## Gaps Identificados e Tratados

- O codigo ativo havia sido revertido de `master`; a implementacao reaproveitavel estava apenas em `origin/gameService`.
- A pasta antiga `voxel-royale/` misturava modulo, gateway e servidor generico; agora a raiz e o monorepo.
- `cmd/server` foi substituido por `services/game`.
- O Gateway nao usa mais `localhost` em container; Compose injeta `GAME_GRPC_ADDR=game:50051`.
- Lobby e um servico separado e containerizado, com salas completas e integracao Lobby->Game (Fase 3).
- Cada servico tem Dockerfile proprio e healthcheck.
- O Game agora mantem estado em memoria para movimentacao validada, abertura de baus, tres armas, dano, eliminacao, safe zone e ranking.

## Desvios em Relacao ao Plano Original

- O codigo gerado ficou em `gen/`, seguindo o que foi reaproveitado da branch revertida, em vez de `internal/contracts/`.
- Alem de `StreamMatch`, o Game expoe `StartMatch` (gRPC interno) chamado pelo Lobby ao iniciar a sala (Fase 3).
