# Papéis e Ownership — Voxel Royale Distribuído

O grupo tem **9 alunos**, organizados em **3 squads** por fatia funcional do sistema
distribuído (não por camada de ferramenta). Não há squad "só de documentação": cada squad
escreve a seção do relatório correspondente ao que implementou.

> **Como preencher:** substitua cada `PLACEHOLDER` pelo nome do aluno e marque o owner de
> cada squad (responsável por revisar PRs e contratos da fatia). Mantenha esta tabela
> sincronizada com a seção "Papéis dos alunos" do relatório (`docs/report/entrega1.tex`).

---

## Squad A — Fluxo canônico de início de partida (Gateway → Lobby → Game)

Dona do caminho que prova os requisitos da Entrega 1 ponta a ponta: criação/entrada de sala,
início (manual e por "todos prontos") e a chamada autoritativa de partida no Game.

| Papel | Aluno | Componentes |
| --- | --- | --- |
| Owner do squad | `PLACEHOLDER` | `internal/lobby`, `proto/lobby/v1`, integração Gateway→Lobby |
| Backend de sala | `PLACEHOLDER` | `LobbyService` (CreateRoom, Join, Start, SetReady) |
| Backend de partida | `PLACEHOLDER` | `internal/game`, `proto/match/v1`, `GameService.StreamMatch` |

Seção do relatório: **Arquitetura** + **Requisitos implementados (gRPC)**.

---

## Squad B — Contrato e esqueleto distribuído (Gateway / contratos / build)

Dona da borda HTTP, dos contratos `.proto`, do código gerado e da consistência entre serviços.

| Papel | Aluno | Componentes |
| --- | --- | --- |
| Owner do squad | `PLACEHOLDER` | `internal/gateway`, grpc-gateway, `gen/` |
| Contratos e geração | `PLACEHOLDER` | `proto/`, Makefile, pipeline `protoc` |
| Web services HTTP | `PLACEHOLDER` | rotas `/v1/rooms/*`, `/v1/match/stream`, `/healthz` |

Seção do relatório: **Requisitos implementados (web services)** + **Detalhes de implementação**.

---

## Squad C — Demo, empacotamento e relatório (Docker / smoke test / SBC)

Dona da reprodutibilidade: Docker Compose, healthchecks, roteiro de demonstração e a
montagem final do relatório SBC a partir das contribuições das outras squads.

| Papel | Aluno | Componentes |
| --- | --- | --- |
| Owner do squad | `PLACEHOLDER` | `deployments/docker-compose.yml`, Dockerfiles |
| Demo e smoke test | `PLACEHOLDER` | README runbook, `Makefile` (demo), validação ponta a ponta |
| Relatório e apresentação | `PLACEHOLDER` | `docs/report/`, consolidação das seções, slides |

Seção do relatório: **Problema** + **Desafios** + consolidação/edição final.

---

## Mapa de ownership por componente

| Componente | Squad responsável |
| --- | --- |
| Gateway (HTTP edge) | B |
| Lobby (salas) | A |
| Game (partida) | A |
| Contratos `.proto` / `gen/` | B |
| Docker Compose / Dockerfiles | C |
| Relatório SBC | C (consolida; cada squad escreve sua seção) |

---

## Processo de mudança de contrato

Qualquer alteração em `proto/lobby/v1` ou `proto/match/v1` (que afeta todas as squads):

1. PR no `.proto` com descrição do impacto e quem consome.
2. Aprovação do owner do Squad B (dono dos contratos) **e** do owner da squad afetada.
3. Regerar `gen/` (`make proto`) e rodar `go test ./...` antes do merge.
