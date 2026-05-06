# Voxel Royale

Monorepo do backend distribuido do Voxel Royale. A estrutura atual separa os tres servicos planejados para a Entrega 1:

- `services/gateway`: entrada HTTP publica e traducao HTTP -> gRPC para o Game.
- `services/game`: servico gRPC autoritativo para movimento, baus, armas, dano, safe zone e ranking.
- `services/lobby`: boilerplate containerizado para a futura gestao de salas.

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

## Requisitos

- Go 1.25.0 ou `GOTOOLCHAIN=auto`.
- Docker 27+ com Docker Compose.
- `protoc` e plugins Go apenas se for regenerar `gen/`.

## Comandos

```powershell
go test ./...
docker compose -f deployments/docker-compose.yml config
docker compose -f deployments/docker-compose.yml up --build
```

## Documentacao

- [Architecture update](docs/architecture.md): arquitetura implementada em relacao ao plano original.
- [Implementation delta](docs/implementation-delta.md): checklist do que foi feito, desvios e gaps restantes.

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
- Lobby agora existe como servico separado e containerizado, ainda em boilerplate.
- Cada servico tem Dockerfile proprio e healthcheck.
- O Game agora mantem estado em memoria para movimentacao validada, abertura de baus, tres armas, dano, eliminacao, safe zone e ranking.

## Desvios em Relacao ao Plano Original

- O plano original esperava Gateway -> Lobby -> Game para sala/partida; como o Lobby foi limitado a boilerplate, o fluxo validado ficou Gateway -> Game.
- O codigo gerado ficou em `gen/`, seguindo o que foi reaproveitado da branch revertida, em vez de `internal/contracts/`.
- O contrato ativo de Game e `StreamMatch`; `StartMatch` fica para a fase em que Lobby iniciar partidas reais.
