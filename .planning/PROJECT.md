# Voxel Royale Distribuido

## What This Is

Voxel Royale Distribuido e um mini battle royale 2D top-down, mobile-first, que roda direto no navegador do celular. Os jogadores entram por QR Code, informam um nome e participam de partidas rapidas de ate 5 minutos com ate 50 jogadores simultaneos.

O projeto e um trabalho pratico de Computacao Distribuida da PUC Minas. A prioridade e demonstrar uma arquitetura distribuida clara, usando Go, microsservicos, gRPC interno, web services, concorrencia, tolerancia a falhas, observabilidade e testes de estresse, sem abrir mao de uma experiencia de jogo realmente jogavel e surpreendente para uma entrega academica.

## Core Value

Demonstrar, de forma jogavel e mensuravel, um sistema distribuido em tempo real no qual 50 jogadores participam de uma partida battle royale voxel com backend Go autoritativo e comunicacao entre servicos via gRPC.

## Requirements

### Validated

(None yet - ship to validate)

### Active

- [ ] Jogador entra por QR Code, informa nome e entra em uma sala sem conta ou autenticacao.
- [ ] Sistema suporta uma partida com 50 jogadores simultaneos, reais ou simulados em teste de carga.
- [ ] Cliente mobile renderiza uma arena 2D top-down em Phaser e oferece controles touch jogaveis.
- [ ] Backend Go e dividido em microsservicos stateless com comunicacao interna via gRPC.
- [ ] Sistema expõe web services para sala, lobby, status, telemetria e operacao.
- [ ] Game server e autoritativo sobre estado de partida, dano, colisao, eliminacao e safe zone.
- [ ] Partida inclui movimentacao, mapa voxel, baus, tres tipos de armas, dano, eliminacao, safe zone e ranking final.
- [ ] Observabilidade mostra chamadas gRPC, tick rate, latencia, uso de rede e comportamento sob carga.
- [ ] Projeto demonstra tolerancia a falhas e prepara a segunda entrega com tratamento explicito de falhas.
- [ ] Infraestrutura roda de forma transportavel com Docker Compose e pode ser implantada em VPS Hostinger.
- [ ] Codigo segue um padrao by-design para que 9 alunos consigam desenvolver em paralelo com consistencia.
- [ ] Primeira entrega demonstra obrigatoriamente gRPC e web services, com relatorio SBC em PDF de ate 4 paginas.

### Out of Scope

- Contas, login, senha ou OAuth - entrada por QR Code + nome e suficiente para o trabalho.
- WebRTC no v1 - WebSocket e mais simples, previsivel e suficiente para tempo real no escopo academico.
- Economia, skins, progresso permanente ou monetizacao - nao ajudam a demonstrar computacao distribuida.
- Backend persistente complexo - servicos devem ser stateless sempre que possivel para portabilidade e escala.
- Grafico voxel proprietario feito do zero - usar assets, sprites e bibliotecas prontas e aceitavel para focar no sistema distribuido.

## Context

- Disciplina: Computacao Distribuida, trabalho pratico em grupo de 7 a 9 pessoas.
- Grupo: 9 alunos; o planejamento deve organizar frentes/times e tornar as responsabilidades explicaveis no relatorio e na apresentacao.
- Tema escolhido: desenvolvimento de um sistema de jogo online distribuido, uma das sugestoes aceitas pelo enunciado.
- Primeira entrega: implementar pelo menos dois requisitos da lista do enunciado. O grupo escolheu gRPC/RPC e web services.
- Primeira entrega tambem exige relatorio em PDF de ate 4 paginas no template SBC, com problema, arquitetura, requisitos implementados, desafios e papel de cada aluno.
- Segunda entrega: expandir a base, implementar tratamento de falhas, escolher pelo menos um requisito adicional, entregar codigo-fonte com instrucoes e apresentar em ate 10 minutos.
- Stack decidida: backend em Go, comunicacao interna gRPC/Protocol Buffers, comunicacao cliente-servidor via WebSocket, frontend com Phaser (2D) + TypeScript, infraestrutura Docker Compose.
- Deploy alvo: Docker Compose transportavel, com caminho claro para VPS Hostinger.

## Constraints

- **Academico**: O sistema deve evidenciar conceitos de computacao distribuida, nao apenas parecer um jogo multiplayer.
- **Entrega 1**: gRPC e web services precisam estar implementados e explicados no relatorio.
- **Jogabilidade**: Mesmo priorizando arquitetura, o jogo deve ser totalmente jogavel para surpreender a banca/turma.
- **Escala demonstravel**: 50 jogadores devem ser suportados; quando nao houver 50 pessoas reais, simuladores de carga devem provar o comportamento.
- **Portabilidade**: O projeto deve subir com Docker Compose em ambiente local e ser implantavel em uma VPS Linux.
- **Equipe**: 9 alunos precisam conseguir trabalhar em paralelo seguindo contratos, padroes e ownership claros.
- **IA e relatorio**: O enunciado permite IA como auxilio, mas textos praticamente gerados automaticamente podem ser desconsiderados; o relatorio deve soar autoral e refletir o entendimento do grupo.

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Go no backend | Concorrencia, rede e gRPC sao pontos fortes do Go e combinam com a disciplina. | - Pending |
| gRPC interno | Atende requisito da Entrega 1 e deixa contratos tipados entre microsservicos. | - Pending |
| Web services HTTP | Atende requisito da Entrega 1 e facilita operacao, lobby, healthchecks e demonstracao. | - Pending |
| WebSocket para tempo real | Menor complexidade que WebRTC para v1 e suficiente para entrada de jogador e broadcast de snapshots. | - Pending |
| Phaser (2D) + TypeScript no cliente | Engine 2D mais simples que um motor 3D para o escopo academico; o backend ja trabalha em coordenadas 2D (x/y). Controles touch e assets prontos. | - Pending |
| Servidor autoritativo | Reduz trapaças e deixa claro quem decide o estado canonico da partida. | - Pending |
| Docker Compose transportavel | Facilita avaliacao, reproducibilidade e deploy em VPS Hostinger. | - Pending |
| Padrao by-design | Necessario para coordenar 9 alunos e manter qualidade academica/top-tier. | - Pending |

---
*Last updated: 2026-04-24 after initialization*
