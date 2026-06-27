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

O cliente tem dois modos automáticos:

- **AO VIVO** — fala com o Gateway real em `POST /v1/match/stream` (via proxy do
  Vite para `:8080`). Suba o backend para isso:
  ```bash
  # em outro terminal, na raiz do repo
  go run ./services/game      # gRPC :50051
  go run ./services/gateway   # HTTP  :8080  (defaults já apontam para localhost)
  ```
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
  main.ts       # bootstrap: DOM (HUD + controles) + Phaser
  GameScene.ts  # cena: loop de input/render, câmera, HUD
  ioRender.ts   # desenho do estilo .io (mapa estático + entidades)
  input.ts      # teclado + joystick/botões touch -> PlayerInput
  net.ts        # LiveDriver (Gateway) + OfflineDriver (mock)
  config.ts     # constantes da arena (espelham server.go) + helpers
  types.ts      # GameState/PlayerInput (espelham match.proto)
  mock.ts       # snapshot mock (fallback offline)
  style.css     # HUD e controles
```

## Próximos passos

- Substituir o request unário por **WebSocket** (Fase 4) para snapshots em tempo real.
- Mira manual e indicação de tiro/dano; animações de morte e loot.
- Tela de lobby (entrar por QR Code) antes da partida.
