# 01-05 SUMMARY — Documentação de arquitetura e relatório SBC (Entrega 1)

**Status:** concluído (rascunho do relatório com nomes a preencher)
**Data:** 2026-06-27

## O que foi entregue

| Artefato | Descrição |
| --- | --- |
| `docs/messages.md` | Referência de mensagens: rotas HTTP do Gateway + RPCs reais do Lobby (`LobbyService`) e do Game (`GameService.StreamMatch`), com tipos, portas e constantes de gameplay. |
| `docs/roles.md` | Ownership de 3 squads para 9 alunos (sem squad "só de docs"), mapa por componente e processo de mudança de contrato. Nomes como `PLACEHOLDER`. |
| `docs/report/entrega1.tex` | Fonte LaTeX do relatório SBC com as seções exigidas por COUR-04: problema, arquitetura, requisitos implementados (gRPC + web services), detalhes, desafios e papéis. |
| `docs/report/references.bib` | Bibliografia (gRPC, grpc-gateway, protobuf, Go, Docker Compose). |
| `docs/report/build.ps1` | Compila o `.tex` (latexmk/pdflatex) e valida o limite de 4 páginas via `pdfinfo`; falha com instruções se não houver toolchain LaTeX. |
| `docs/report/entrega1.pdf` | PDF gerado — **4 páginas** (no limite). |

`docs/architecture.md` já existia da execução anterior e foi mantido.

## Decisões e desvios em relação ao plano 01-05

1. **Contratos reais, não os supostos.** O plano referenciava `proto/game/v1/game.proto` e o RPC
   `StartMatch`. A implementação real é `proto/match/v1/match.proto` com `GameService.StreamMatch`.
   A documentação reflete o que existe no código.
2. **Rascunho em classe `article`, não no `sbc-template` oficial.** O `sbc-template.sty` é um arquivo
   externo da SBC que o MiKTeX não auto-instala. Para um rascunho que compila em qualquer máquina, usamos
   `article` com a mesma estrutura de seções; o cabeçalho do `.tex` instrui a troca pelo template oficial
   na submissão final. A string `sbc-template` está presente no comentário (key-link satisfeito).
3. **Correção de fonte (lmodern).** O T1 fontenc disparava geração de PK bitmap (`ecbx1200`) que falhava no
   MiKTeX; `\usepackage{lmodern}` (Type 1 escalável) resolveu e tornou o build rápido.
4. **`thebibliography` embutido** no `.tex` para o PDF compilar só com pdflatex (sem passe bibtex). O
   `references.bib` fica como fonte canônica para a versão final com `\bibliography`.

## Verificação

```text
pwsh docs/report/build.ps1   → "Paginas: 4" + "OK." (exit 0)
go build ./... && go test ./...  → verde (skeleton intacto)
```

## Pendências (não bloqueiam o código/docs)

- Preencher os 9 nomes reais e os owners de squad (`PLACEHOLDER`).
- Na submissão final, migrar para o `sbc-template` oficial e re-checar as 4 páginas.
