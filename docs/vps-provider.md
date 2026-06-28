# VPS Provider Setup

Esta e a Fase 8 do roadmap. Ela existe para separar o que ja esta pronto no
repositorio (Compose local, readiness e guia de deploy) da configuracao de uma
VPS real, que depende de conta, credenciais e acesso de rede.

## Decisao do Provedor

Opcoes aceitaveis:

- Hostinger VPS, se for o provedor definido pelo grupo.
- Outro provedor equivalente (DigitalOcean, Oracle Cloud, AWS Lightsail, etc.) se
  o grupo preferir custo/acesso diferente.

Requisitos minimos recomendados para a demo:

- 2 vCPU.
- 2 GB RAM.
- 20 GB disco.
- Ubuntu 22.04/24.04 LTS.
- Acesso SSH com usuario sudo.
- Portas liberaveis: `5173`, `8080`, `3000`, `9090`, `16686` para demo direta,
  ou `80/443` se houver proxy/reverse proxy.

## Informacoes Necessarias

Antes de executar o deploy remoto, registrar:

- Provedor escolhido:
- Plano/tamanho da VPS:
- Sistema operacional:
- IP publico:
- Dominio, se houver:
- Usuario SSH:
- Metodo de autenticacao: chave SSH ou senha:
- Portas liberadas no firewall do provedor:
- Docker/Compose ja instalado? sim/nao:

## Checklist de Configuracao

- [ ] Criar/confirmar conta no provedor.
- [ ] Provisionar VPS com Ubuntu LTS.
- [ ] Configurar SSH e testar login.
- [ ] Atualizar sistema (`apt update && apt upgrade`).
- [ ] Instalar Docker Engine e Docker Compose v2.
- [ ] Liberar portas no firewall do provedor e no `ufw`, se ativo.
- [ ] Clonar o repositorio na VPS.
- [ ] Rodar `docker compose -f deployments/docker-compose.yml up --build -d`.
- [ ] Validar `http://<ip>:5173/frontend-healthz`.
- [ ] Validar `http://<ip>:8080/readyz`.
- [ ] Validar `http://<ip>:8080/metrics`.
- [ ] Rodar `go run ./tools/stress50 -gateway http://<ip>:8080 -players 50 -duration 30s`.
- [ ] Registrar saida do stress remoto em `docs/stress-results.md`.

## Resultado Esperado

A Fase 8 so deve ser marcada como concluida quando houver evidencia de endpoints
publicos funcionando na VPS real e resultado de stress remoto capturado.
