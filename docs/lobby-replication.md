# Replicacao Primario-Backup do Lobby

O Lobby agora implementa um algoritmo simples de controle de replicas no modelo
primario-backup. A replicacao cobre o estado de salas, jogadores, dono da sala,
status, flags de pronto e contadores internos usados para gerar IDs.

## Modelo

- `lobby-primary` recebe as operacoes de escrita usadas pelo Gateway.
- `lobby-backup` recebe snapshots versionados por um endpoint interno HTTP.
- Cada escrita bem-sucedida no primario incrementa uma versao monotonicamente.
- O backup so aceita o evento se `version == lastVersion + 1`.
- Eventos duplicados com a versao atual sao tratados como idempotentes somente
  se carregarem o mesmo snapshot ja aplicado.
- Eventos fora de ordem sao rejeitados para evitar divergencia silenciosa.

As operacoes cobertas sao:

- `CreateRoom`
- `JoinRoom`
- `LeaveRoom`
- `SetReady`
- `StartRoom`

## Caminho de Escrita

1. O primario valida a operacao.
2. O primario aplica a mudanca em memoria.
3. O primario gera um snapshot completo do estado do Lobby.
4. O primario envia o snapshot ao backup em `/replication/lobby-state`.
5. O backup aplica o snapshot somente se a versao estiver na ordem esperada.
6. O primario responde sucesso ao cliente apenas depois da confirmacao do backup.

Se o backup nao confirmar, o primario desfaz a escrita local e retorna erro. Isso
mantem a propriedade de que uma escrita confirmada pelo Lobby tambem foi aplicada
na replica.

## Topologia Docker

O Compose sobe dois Lobbies:

- `lobby-primary`: `LOBBY_REPLICATION_ROLE=primary`
- `lobby-backup`: `LOBBY_REPLICATION_ROLE=backup`

O Gateway continua apontando para `lobby-primary:50052`. O endpoint de replicacao
fica disponivel apenas dentro da rede Docker:

```text
http://lobby-backup:8081/replication/lobby-state
```

O Gateway tambem conhece o backup e sua rota interna de promocao:

```text
LOBBY_BACKUP_GRPC_ADDR=lobby-backup:50052
LOBBY_BACKUP_PROMOTE_URL=http://lobby-backup:8081/replication/promote
```

## Failover

Quando uma chamada do Gateway para o Lobby primario falha com erro de
disponibilidade (`Unavailable` ou `DeadlineExceeded`), o Gateway:

1. Chama `POST /replication/promote` no `lobby-backup`.
2. O backup muda de papel e passa a aceitar escritas publicas do `LobbyService`.
3. O Gateway troca seu endpoint ativo para `lobby-backup:50052`.
4. A operacao que falhou e repetida uma vez no backup promovido.
5. As proximas operacoes de sala seguem diretamente para o backup promovido.

Essa troca e intencionalmente aderente: depois que o Gateway promove o backup,
ele nao volta automaticamente para o primario antigo. Isso reduz o risco de
split-brain se o primario antigo voltar com estado desatualizado.

## Metricas

Foram adicionadas metricas Prometheus:

- `voxel_lobby_replication_events_total{operation,result}`
- `voxel_lobby_replication_version`
- `voxel_gateway_lobby_failovers_total{operation,result}`

O Prometheus coleta tanto `lobby-primary:8081` quanto `lobby-backup:8081`.

## Escopo e Limitacoes

A replicacao foi aplicada ao Lobby porque ele guarda o estado de coordenacao das
salas. O processamento em tempo real da partida continua no Game e nao passa a
ser replicado por esta mudanca.

Antes da promocao, o backup e read-only para chamadas publicas do Lobby. Ele
aplica apenas eventos vindos do canal interno de replicacao. Depois da promocao,
ele passa a operar como novo primario sem uma segunda replica a jusante.
