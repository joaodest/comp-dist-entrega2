# Sistema de Desenvolvimento em Equipe

Este guia implementa a Fase 2 do roadmap: permitir que 9 alunos trabalhem em
paralelo sem quebrar contratos, ownership ou arquitetura.

## Objetivo da Fase 2

A regra principal e simples: cada mudanca deve deixar claro qual contrato toca,
qual servico e dono do comportamento e quais validacoes provam que nada quebrou.

Este documento cobre o requisito `ARCH-06`:

- estrutura de pastas;
- ownership;
- contratos;
- testes;
- regras de contribuicao;
- divisao de tarefas por frente.

## Estrutura by-design

```text
.
├── proto/                  # contratos fonte; mudancas exigem review cruzado
├── gen/                    # codigo gerado; nao editar manualmente
├── internal/
│   ├── gateway/            # HTTP edge e grpc-gateway
│   ├── lobby/              # ciclo de vida de salas
│   └── game/               # estado autoritativo de partida
├── services/
│   ├── gateway/            # entrypoint e Dockerfile do Gateway
│   ├── lobby/              # entrypoint e Dockerfile do Lobby
│   └── game/               # entrypoint e Dockerfile do Game
├── deployments/            # Compose e configuracao de runtime local
├── frontend/               # cliente Phaser/Vite
└── docs/                   # arquitetura, mensagens, papeis e relatorios
```

## Fronteiras dos servicos

### Gateway

Responsabilidade:

- expor HTTP publico;
- registrar rotas grpc-gateway;
- manter `/healthz`;
- traduzir chamadas HTTP/JSON para gRPC interno.

Nao deve:

- guardar estado de sala ou partida;
- duplicar regras de negocio do Lobby ou Game;
- conhecer detalhes de balanceamento de armas ou safe zone.

### Lobby

Responsabilidade:

- criar, consultar e fechar salas;
- controlar jogadores, owner e estado `ready`;
- validar capacidade, status e permissao de inicio;
- futuramente iniciar partida real no Game.

Nao deve:

- calcular dano, movimento ou ranking;
- manter estado de gameplay;
- expor HTTP diretamente.

### Game

Responsabilidade:

- manter estado autoritativo de partida;
- validar input de jogador;
- aplicar movimento, bau, armas, dano, safe zone e ranking;
- devolver snapshots consistentes.

Nao deve:

- criar salas ou QR Code;
- assumir regras de UI;
- depender de estado global do Gateway.

### Frontend

Responsabilidade:

- renderizar estado recebido do servidor;
- capturar teclado/touch;
- manter fallback offline apenas como demo;
- respeitar os tipos JSON derivados dos contratos protobuf.

Nao deve:

- decidir dano, eliminacao ou ranking;
- depender de campos que nao existem em `proto/`;
- mascarar erro de contrato sem atualizar docs/testes.

## Ownership

O ownership oficial fica em `docs/roles.md`. A regra de review e:

- arquivos de um servico precisam de review do squad dono daquele servico;
- arquivos `proto/` precisam de review do Squad B e da squad consumidora;
- mudancas que atravessam Gateway, Lobby e Game precisam de pelo menos dois
  owners;
- alteracoes no relatorio devem citar a squad que implementou a parte descrita.

Enquanto os nomes reais estiverem como `PLACEHOLDER`, os squads ainda podem
trabalhar por area. Antes da entrega, cada placeholder deve virar um aluno real.

## Contratos e compatibilidade

Contratos publicos do projeto:

- HTTP Gateway documentado em `docs/messages.md`;
- gRPC fonte em `proto/lobby/v1/lobby.proto` e `proto/match/v1/match.proto`;
- JSON consumido pelo frontend em camelCase, gerado pelo grpc-gateway.

Regras:

1. Nunca editar `gen/` manualmente.
2. Toda mudanca em `proto/` deve atualizar `docs/messages.md`.
3. Toda mudanca em `proto/` deve regenerar `gen/` com `make proto`.
4. Remover ou renomear campo exige atualizar consumidores no mesmo PR.
5. Campos novos devem ter comportamento documentado e teste quando afetarem regra
   de negocio.
6. Se uma mudanca prepara fase futura, ela deve ficar descrita como parcial ou
   pendente; nao esconder lacuna em fallback silencioso.

## Testes esperados

Backend:

- `internal/lobby`: testes para ciclo de sala, limites, ownership, ready e
  erros de status.
- `internal/game`: testes para input, movimento, armas, dano, safe zone e
  ranking.
- `internal/gateway`: testes para config, healthcheck e registro de rotas quando
  o handler mudar.

Frontend:

- hoje nao ha suite automatizada dedicada;
- toda mudanca em `frontend/` deve passar por `npm --prefix frontend run build`;
- quando a tela de lobby/WebSocket entrar, adicionar testes ou checks de contrato
  passa a ser criterio da fase.

Infra:

- mudancas em Dockerfile, Compose, env vars ou portas exigem
  `docker compose -f deployments/docker-compose.yml config`;
- se Docker Desktop estiver disponivel, validar tambem `up --build`.

## Divisao de tarefas por frente

As proximas fases podem ser paralelizadas assim:

- Backend Lobby/Game: integrar `StartRoom` ao Game, criar partidas por sala e
  definir lifecycle de match.
- Gateway/Realtime: adicionar WebSocket, correlacao de logs e fanout de snapshot.
- Frontend: tela de lobby, QR Code, ranking final, feedback visual de tiro/dano.
- Observabilidade: traces, metricas, dashboards e naming padrao de spans.
- Carga: simulador de 50 jogadores e roteiro de coleta de resultados.
- Infra/Deploy: Compose completo com frontend/telemetria e runbook VPS.
- Relatorio: manter papel dos alunos, resultados e tradeoffs sempre alinhados ao
  codigo real.

## Checklist para escolher uma tarefa

Antes de pegar uma tarefa, o aluno deve responder:

1. Qual requisito do roadmap esta sendo atendido?
2. Qual squad e dona dos arquivos?
3. Existe contrato `.proto` ou HTTP envolvido?
4. Qual teste ou build prova a mudanca?
5. Qual trecho de documentacao precisa ser atualizado?

Se duas ou mais respostas ficarem indefinidas, a tarefa ainda precisa de plano
antes de virar codigo.

## Definition of Done

Uma mudanca esta pronta quando:

- compila ou passa no build relevante;
- tem teste quando altera regra de negocio;
- atualiza docs de contrato quando muda API;
- nao quebra ownership definido em `docs/roles.md`;
- deixa claro o que ainda e gap de fase futura.

