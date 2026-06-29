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

## Metricas

Foram adicionadas metricas Prometheus:

- `voxel_lobby_replication_events_total{operation,result}`
- `voxel_lobby_replication_version`

O Prometheus coleta tanto `lobby-primary:8081` quanto `lobby-backup:8081`.

## Escopo e Limitacoes

A replicacao foi aplicada ao Lobby porque ele guarda o estado de coordenacao das
salas. O processamento em tempo real da partida continua no Game e nao passa a
ser replicado por esta mudanca.

O backup e read-only para chamadas publicas do Lobby. Ele aplica apenas eventos
vindos do canal interno de replicacao. Portanto, esta implementacao demonstra
controle de replicas e consistencia entre primario e backup, mas ainda nao faz
failover automatico transparente do Gateway para o backup.
