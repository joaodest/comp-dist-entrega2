# Deploy Local e VPS

Este guia cobre a Fase 7: subir o sistema completo com Docker Compose, validar
readiness e repetir os testes de falha/carga para a Entrega 2. A configuracao do
provedor VPS real e a validacao remota foram separadas para a Fase 8 em
[`vps-provider.md`](vps-provider.md).

## Serviços e Portas

- Frontend/Nginx: `http://localhost:5173`
- Gateway HTTP/WebSocket: `http://localhost:8080`
- Prometheus: `http://localhost:9090`
- Grafana: `http://localhost:3000/d/voxel-royale/voxel-royale`
- Jaeger: `http://localhost:16686/search`
- Game gRPC interno: `game:50051`
- Lobby primario gRPC interno: `lobby-primary:50052`
- Lobby backup gRPC interno: `lobby-backup:50052`
- Replicacao interna do Lobby: `http://lobby-backup:8081/replication/lobby-state`

## Execução Local

```bash
docker compose -f deployments/docker-compose.yml config
docker compose -f deployments/docker-compose.yml up --build
```

Valide readiness:

```bash
curl http://localhost:5173/frontend-healthz
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
curl http://localhost:8080/metrics
```

Rode a prova de 50 jogadores:

```bash
make stress50
```

## Deploy em VPS Hostinger ou Equivalente

Este roteiro e reproduzivel, mas ainda depende de uma VPS real configurada. Antes
de executar, preencha o checklist de provedor em [`vps-provider.md`](vps-provider.md).

Pré-requisitos na VPS:

- Docker Engine com Compose v2.
- Portas liberadas no firewall: `80`/`443` se usar proxy externo, ou as portas
  diretas `5173`, `8080`, `9090`, `3000`, `16686` para demonstração acadêmica.
- Pelo menos 2 vCPU e 2 GB de RAM para demo com frontend, backends e telemetria.

Passos:

```bash
git clone <repo-url> voxel-royale
cd voxel-royale
docker compose -f deployments/docker-compose.yml pull
docker compose -f deployments/docker-compose.yml up --build -d
docker compose -f deployments/docker-compose.yml ps
```

Validação na VPS:

```bash
curl http://<ip-ou-dominio>:5173/frontend-healthz
curl http://<ip-ou-dominio>:8080/readyz
curl http://<ip-ou-dominio>:8080/metrics
go run ./tools/stress50 -gateway http://<ip-ou-dominio>:8080 -players 50 -duration 30s
```

## Readiness e Falhas Controladas

- `/healthz` indica que o processo HTTP do serviço está vivo.
- `/readyz` indica que o serviço está pronto para receber tráfego.
- Gateway só fica ready quando consegue abrir TCP para Game e Lobby primario.
- Lobby primario e Lobby backup ficam ready quando seu gRPC esta ouvindo e o Game esta acessivel.
- Game fica ready quando o listener gRPC foi aberto.

Para demonstrar degradação controlada:

```bash
docker compose -f deployments/docker-compose.yml stop game
curl -i http://localhost:8080/readyz
docker compose -f deployments/docker-compose.yml start game
```

O Gateway/Lobby devem responder `503` em `/readyz` enquanto a dependência estiver
fora, sem derrubar o processo. O teste de unidade do Lobby também cobre a falha
de `StartMatch`: a sala volta para `WAITING` quando o Game não confirma a partida.

Para demonstrar a replicacao primario-backup do Lobby, crie uma sala pelo Gateway
e consulte as metricas `voxel_lobby_replication_version` e
`voxel_lobby_replication_events_total` no Prometheus. Se o `lobby-backup` estiver
fora, escritas no `lobby-primary` retornam erro em vez de confirmar uma sala que
nao foi replicada.

## Estado e Stateless

Gateway e frontend são stateless. O estado de salas do Lobby fica em memoria e e
replicado do primario para o backup por snapshots versionados. O Game ainda mantem
partidas em memoria sem replicacao, suficiente para o escopo academico de partidas
efemeras; reiniciar o Game perde partidas atuais, mas nao corrompe dados
persistentes porque o projeto nao usa persistencia duravel nesta entrega.
