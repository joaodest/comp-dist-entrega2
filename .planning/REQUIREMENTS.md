# Requirements: Voxel Royale Distribuido

**Defined:** 2026-04-24  
**Core Value:** Demonstrar, de forma jogavel e mensuravel, um sistema distribuido em tempo real no qual 50 jogadores participam de uma partida battle royale voxel com backend Go autoritativo e comunicacao entre servicos via gRPC.

## v1 Requirements

Requirements for the semester project and its first two assessed deliveries. Each maps to roadmap phases.

### Course Delivery

- [ ] **COUR-01**: Grupo consegue explicar o problema escolhido como jogo online distribuido no contexto de Computacao Distribuida.
- [ ] **COUR-02**: Primeira entrega implementa gRPC/RPC como requisito obrigatorio.
- [ ] **COUR-03**: Primeira entrega implementa web services como requisito obrigatorio.
- [ ] **COUR-04**: Primeira entrega inclui relatorio PDF de ate 4 paginas no template SBC com problema, arquitetura, requisitos implementados, desafios e papel de cada aluno.
- [ ] **COUR-05**: Segunda entrega inclui codigo-fonte com instrucoes de execucao e apresentacao de ate 10 minutos.
- [ ] **COUR-06**: Cada aluno tem papel documentado e consegue responder perguntas sobre sua parte.

### Architecture

- [ ] **ARCH-01**: Sistema roda como pelo menos dois nos/servicos distribuidos conectados em rede.
- [ ] **ARCH-02**: Backend e composto por microsservicos Go separados para Gateway, Lobby e Game.
- [ ] **ARCH-03**: Contratos `.proto` definem as mensagens gRPC principais entre servicos.
- [ ] **ARCH-04**: Servicos publicos expõem web services HTTP para criacao/entrada/status de sala e healthchecks.
- [ ] **ARCH-05**: Arquitetura documenta quem troca mensagem com quem, o conteudo das principais mensagens e o papel de cada entidade.
- [ ] **ARCH-06**: Padrao by-design define estrutura de pastas, ownership, contratos, testes e regras para contribuicao dos 9 alunos.

### Lobby and Session

- [ ] **LOBB-01**: Usuario consegue criar uma sala e obter um QR Code/URL com token unico.
- [ ] **LOBB-02**: Jogador consegue entrar na sala pelo celular usando QR Code/URL e informando apenas nome.
- [ ] **LOBB-03**: Lobby mostra jogadores conectados e estado de pronto/aguardando.
- [ ] **LOBB-04**: Lobby inicia a partida quando criterio de inicio e atingido por pronto manual ou limite de tempo.

### Realtime Networking

- [ ] **NETW-01**: Cliente mantem conexao WebSocket com o Gateway durante a partida.
- [ ] **NETW-02**: Cliente envia inputs de movimento e acoes com sequencia/timestamp suficiente para reconciliacao.
- [ ] **NETW-03**: Gateway encaminha inputs ao Game service via gRPC.
- [ ] **NETW-04**: Game service envia snapshots de estado para o Gateway distribuir aos clientes.
- [ ] **NETW-05**: Sistema suporta ate 50 jogadores simultaneos em uma partida real ou simulada.

### Gameplay

- [ ] **GAME-01**: Cliente renderiza uma arena 2D top-down em Phaser no navegador mobile.
- [ ] **GAME-02**: Jogador consegue mover o personagem com controles touch.
- [ ] **GAME-03**: Game service gera spawns de jogadores, baus e armas.
- [ ] **GAME-04**: Jogador consegue abrir baus e coletar armas.
- [ ] **GAME-05**: Jogo possui tres tipos de armas com comportamentos distintos.
- [ ] **GAME-06**: Game service valida dano, vida e eliminacao de jogadores.
- [ ] **GAME-07**: Safe zone encolhe ao longo da partida para forcar termino em ate 5 minutos.
- [ ] **GAME-08**: Partida termina com ranking final por sobrevivencia e desempenho.

### Observability and Scale

- [ ] **OBSV-01**: Servicos emitem traces OpenTelemetry para chamadas HTTP, WebSocket lifecycle e gRPC.
- [ ] **OBSV-02**: Prometheus coleta metricas de tick rate, latencia gRPC, jogadores conectados, banda/payload e erros.
- [ ] **OBSV-03**: Grafana ou Jaeger permite demonstrar visualmente o fluxo Gateway -> Lobby/Game.
- [ ] **OBSV-04**: Teste de estresse simula 50 jogadores conectando, enviando inputs e recebendo snapshots.
- [ ] **OBSV-05**: Resultado do teste de estresse e registrado para embasar escalabilidade no relatorio/apresentacao.

### Fault Tolerance and Deploy

- [ ] **FAIL-01**: Sistema lida com desconexao de jogador sem derrubar a partida.
- [ ] **FAIL-02**: Servicos detectam falha de comunicacao gRPC e retornam erro controlado/estado degradado.
- [ ] **FAIL-03**: Segunda entrega implementa metodos adequados para tratamento de falhas e testes correspondentes.
- [ ] **DEPL-01**: Todo o sistema sobe localmente com Docker Compose.
- [ ] **DEPL-02**: Deploy em VPS Hostinger tem instrucoes reproduziveis.
- [ ] **DEPL-03**: Servicos sao stateless sempre que possivel e expõem health/readiness checks.

## v2 Requirements

Deferred beyond the assessed course scope unless time remains.

### Advanced Distributed Algorithms

- **DIST-01**: Implementar eleicao de lider entre game workers.
- **DIST-02**: Implementar relogios logicos para ordenar eventos de partida.
- **DIST-03**: Implementar controle de replicas ou consistencia distribuida para match state.
- **DIST-04**: Implementar Two-Phase Commit para transacoes distribuidas.

### Product Extras

- **PROD-01**: Usuario cria conta persistente.
- **PROD-02**: Jogador mantem historico permanente de partidas.
- **PROD-03**: Jogador desbloqueia skins ou progresso permanente.
- **PROD-04**: Sistema oferece matchmaking publico fora de salas por QR Code.

## Out of Scope

| Feature | Reason |
|---------|--------|
| Login/autenticacao completa | QR Code + nome atende o escopo e evita complexidade irrelevante. |
| WebRTC | WebSocket e suficiente e reduz risco para Entrega 1. |
| Kubernetes | Docker Compose atende portabilidade e evita overhead academico. |
| Graficos complexos ou assets originais | Usar assets prontos preserva foco em sistemas distribuidos. |
| Persistencia duravel de progresso | Partidas efemeras bastam para demonstrar arquitetura. |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| COUR-01 | Phase 1 | Pending |
| COUR-02 | Phase 1 | Pending |
| COUR-03 | Phase 1 | Pending |
| COUR-04 | Phase 1 | Pending |
| COUR-05 | Phase 8 | Pending |
| COUR-06 | Phase 8 | Pending |
| ARCH-01 | Phase 1 | Pending |
| ARCH-02 | Phase 1 | Pending |
| ARCH-03 | Phase 1 | Pending |
| ARCH-04 | Phase 1 | Pending |
| ARCH-05 | Phase 1 | Pending |
| ARCH-06 | Phase 2 | Pending |
| LOBB-01 | Phase 3 | Pending |
| LOBB-02 | Phase 3 | Pending |
| LOBB-03 | Phase 3 | Pending |
| LOBB-04 | Phase 3 | Pending |
| NETW-01 | Phase 4 | Pending |
| NETW-02 | Phase 4 | Pending |
| NETW-03 | Phase 4 | Pending |
| NETW-04 | Phase 4 | Pending |
| NETW-05 | Phase 6 | Pending |
| GAME-01 | Phase 5 | Pending |
| GAME-02 | Phase 5 | Pending |
| GAME-03 | Phase 5 | Pending |
| GAME-04 | Phase 5 | Pending |
| GAME-05 | Phase 5 | Pending |
| GAME-06 | Phase 5 | Pending |
| GAME-07 | Phase 5 | Pending |
| GAME-08 | Phase 5 | Pending |
| OBSV-01 | Phase 6 | Pending |
| OBSV-02 | Phase 6 | Pending |
| OBSV-03 | Phase 6 | Pending |
| OBSV-04 | Phase 6 | Pending |
| OBSV-05 | Phase 6 | Pending |
| FAIL-01 | Phase 7 | Pending |
| FAIL-02 | Phase 7 | Pending |
| FAIL-03 | Phase 7 | Pending |
| DEPL-01 | Phase 7 | Pending |
| DEPL-02 | Phase 7 | Pending |
| DEPL-03 | Phase 7 | Pending |

**Coverage:**
- v1 requirements: 40 total
- Mapped to phases: 40
- Unmapped: 0

---
*Requirements defined: 2026-04-24*
*Last updated: 2026-04-24 after roadmap creation*
