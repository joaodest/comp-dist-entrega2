# Guia de Contribuicao

Este repositorio e trabalhado por 9 alunos em paralelo. Antes de abrir ou revisar
uma mudanca, use este guia junto com `docs/team-development.md` e
`docs/roles.md`.

## Fluxo recomendado

1. Atualize sua branch a partir de `master`.
2. Entenda qual squad e dona dos arquivos que voce vai alterar em
   `docs/roles.md`.
3. Mantenha a mudanca pequena: um fluxo, contrato ou componente por PR.
4. Rode as validacoes locais antes de pedir review.
5. Descreva no PR quais servicos, rotas, RPCs ou telas foram afetados.

## Validacoes locais

Backend:

```bash
go test ./...
```

Frontend:

```bash
npm --prefix frontend ci
npm --prefix frontend run build
```

Docker, quando a mudanca afetar empacotamento ou configuracao de servico:

```bash
docker compose -f deployments/docker-compose.yml config
docker compose -f deployments/docker-compose.yml up --build
```

`make proto` so deve ser usado quando um arquivo em `proto/` mudar. Ele exige
`protoc`, plugins Go e `third_party/googleapis`.

## Regras de mudanca por area

- `proto/**`: precisa de review do Squad B e da squad consumidora; gere `gen/`
  no mesmo PR.
- `internal/gateway/**`: confirme rotas HTTP, proxy grpc-gateway e healthcheck.
- `internal/lobby/**`: preserve isolamento entre salas, ownership e validacoes
  de capacidade/status.
- `internal/game/**`: preserve autoridade do servidor, validacao de input e
  determinismo suficiente para testes.
- `frontend/**`: mantenha tipos alinhados ao JSON gerado pelos contratos protobuf.
- `deployments/**` e `services/**/Dockerfile`: valide Compose e healthchecks.
- `docs/report/**`: mantenha consistencia com `docs/roles.md` e limite da entrega.

## Checklist de PR

- [ ] A mudanca tem escopo claro e nao mistura refactor com feature.
- [ ] Contratos alterados foram documentados em `docs/messages.md`.
- [ ] Testes Go foram adicionados ou atualizados quando a regra de negocio mudou.
- [ ] Build do frontend foi executado quando `frontend/` mudou.
- [ ] Docker Compose foi validado quando portas, env vars ou Dockerfiles mudaram.
- [ ] Owners afetados foram marcados para review.

