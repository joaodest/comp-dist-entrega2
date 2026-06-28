# Observabilidade e Prova de 50 Jogadores

Esta fase torna mensuravel o fluxo distribuido Browser/WebSocket -> Gateway -> Game
e Lobby -> Game. A stack local sobe Prometheus, Grafana e Jaeger junto com os tres
servicos Go.

## Como Rodar

```bash
docker compose -f deployments/docker-compose.yml up --build
```

Endpoints principais:

- Gateway metrics: http://localhost:8080/metrics
- Prometheus: http://localhost:9090
- Grafana: http://localhost:3000/d/voxel-royale/voxel-royale
- Jaeger: http://localhost:16686/search

## Metricas Prometheus

O Gateway mede sessoes WebSocket, mensagens/bytes por direcao, erros do pipeline
realtime e latencia da chamada `PushInput` para o Game.

O Game mede inputs aceitos/rejeitados, streams `WatchMatch`, partidas, jogadores,
assinantes, ticks processados, duracao do tick e snapshots descartados por cliente
lento.

O Lobby mede salas, jogadores em sala e eventos de sala (`create`, `join`, `start`,
`ready`, `leave`).

## Traces OpenTelemetry

Os servicos usam OpenTelemetry para instrumentar entrada HTTP e chamadas gRPC. No
Compose, `OTEL_EXPORTER_OTLP_ENDPOINT=http://jaeger:4318` envia spans para o Jaeger.
Procure pelos servicos `voxel-gateway`, `voxel-lobby` e `voxel-game`.

## Teste de Estresse

Com a stack ativa, rode:

```bash
make stress50
```

Ou ajuste parametros:

```bash
go run ./tools/stress50 -gateway http://localhost:8080 -players 50 -duration 30s
```

O runner cria uma sala com capacidade 50, entra com bots, inicia a partida pelo
Lobby, abre um WebSocket por jogador, envia inputs a cada ~66 ms e imprime um JSON
com jogadores conectados, inputs enviados, snapshots recebidos, bytes e erros.

Use o JSON emitido pelo runner junto com o dashboard Grafana para registrar o
resultado no relatorio/apresentacao.

O smoke local registrado em [`stress-results.md`](stress-results.md) validou 50
WebSockets simultaneos por 5s, com 3.750 inputs enviados, 3.750 snapshots recebidos
e zero erros do simulador.
