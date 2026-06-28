# Voxel Royale — Frontend (Phaser 2D, estilo .io)

Cliente do jogo em **Phaser 3 + TypeScript + Vite**, no estilo top-down cartoon (.io):
mapa com grama/rio/árvores/pedras, jogador controlável, zona segura, baús e
demais jogadores renderizados a partir do `GameState` autoritativo do backend.

## Rodar

```bash
cd frontend
npm install
npm run dev      # http://localhost:5173
```

Ao abrir, o cliente mostra a **tela de lobby** (Fase 3): criar sala com QR
Code/URL, entrar por nome (também via `?room=<id>`), marcar pronto e iniciar a
partida. Suba os três serviços para o fluxo completo:

```bash
# em outro terminal, na raiz do repo
go run ./services/game      # gRPC :50051
go run ./services/lobby     # gRPC :50052 (disca GAME_GRPC_ADDR)
go run ./services/gateway   # HTTP  :8080  (defaults já apontam para localhost)
```

Durante a partida, o cliente tem dois modos automáticos:

- **AO VIVO** — fala com o Gateway em `POST /v1/match/stream` (via proxy do Vite
  para `:8080`), enviando o `roomId` da sala.
- **OFFLINE (mock)** — se o Gateway não responder, o cliente cai num simulador
  local para continuar jogável (badge muda para "OFFLINE").

## Controles

- **Mover:** WASD / setas, ou o **joystick** (canto inferior esquerdo, touch/mouse).
- **Atacar:** espaço ou botão **ATIRAR** (auto-mira no inimigo mais próximo no alcance).
- **Abrir baú:** tecla **E** ou botão **BAÚ** (abre o baú mais próximo no alcance).

## Como funciona

Um loop envia o input do jogador ao Gateway a cada ~90 ms (`SEND_MS`); o Game é
autoritativo (movimento, dano, baús, zona, ranking) e devolve o `GameState`, que o
cliente interpola e desenha. A câmera segue o seu jogador (`player-web`).

## Estrutura

```
src/
  main.ts       # bootstrap: roda o lobby e, ao iniciar, monta HUD + Phaser
  lobby.ts      # cliente HTTP da API de salas (/v1/rooms/*)
  lobbyUI.ts    # telas de lobby: criar/entrar/QR/ready/start
  session.ts    # sessão (roomId/myId/isOwner) usada pela partida
  GameScene.ts  # cena: loop de input/render, câmera, HUD
  ioRender.ts   # desenho do estilo .io (mapa estático + entidades)
  input.ts      # teclado + joystick/botões touch -> PlayerInput
  net.ts        # LiveDriver (Gateway) + OfflineDriver (mock)
  config.ts     # constantes da arena (espelham server.go) + helpers
  types.ts      # GameState/PlayerInput (espelham match.proto)
  mock.ts       # snapshot mock (fallback offline)
  style.css     # lobby, HUD e controles
```

## Próximos passos

- Substituir o request unário por **WebSocket** (Fase 4) para snapshots em tempo real.
- Mira manual e indicação de tiro/dano; animações de morte e loot.
- Tela de fim de partida com ranking (Fase 5).
