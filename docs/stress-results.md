# Stress Results

## 2026-06-28 — Local 50-Player Smoke

Comando:

```bash
go run ./tools/stress50 -gateway http://localhost:8080 -players 50 -duration 5s
```

Ambiente:

- Serviços Go locais: Game `:50051`/`:8082`, Lobby `:50052`/`:8081`, Gateway `:8080`.
- Duração curta para smoke de desenvolvimento; para apresentação, repetir com `-duration 30s` usando a stack Docker com Prometheus/Grafana/Jaeger.

Resultado:

```json
{
  "avgSnapshotsPerPlayer": 75,
  "durationSeconds": 5.0015982,
  "inputsSent": 3750,
  "playersConnected": 50,
  "playersRequested": 50,
  "roomId": "room-1",
  "simulatorErrors": 0,
  "snapshotBytesReceived": 63700650,
  "snapshotsPerSecond": 749.7603466028119,
  "snapshotsReceived": 3750
}
```

Leitura:

- `NETW-05` validado em smoke: 50 conexões WebSocket simultâneas em uma partida simulada.
- Cada bot recebeu ~75 snapshots em 5s, coerente com o relógio de 15 Hz do Game.
- O runner não observou erros durante a simulação.
