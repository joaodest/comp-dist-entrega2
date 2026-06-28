import './style.css';
import Phaser from 'phaser';
import { GameScene } from './GameScene';
import { Input } from './input';
import { runLobby } from './lobbyUI';

const app = document.getElementById('app')!;

function startGame() {
  app.innerHTML = `
    <div id="game"></div>
    <div id="hud">
      <div class="hud__top">
        <div class="hud__brand">
          <span class="hud__dot"></span> Voxel Royale
          <span id="mode" class="badge badge--off">conectando…</span>
        </div>
        <div class="hud__stats">
          <span class="stat"><b id="alive">—</b> vivos</span>
          <span class="stat">fase <b id="phase">—</b></span>
          <span class="stat">tick <b id="tick">—</b></span>
        </div>
      </div>
      <div class="hud__me">
        <span class="chip">VIDA <b id="hp">—</b></span>
        <span class="chip">ARMA <b id="weapon">—</b></span>
      </div>
      <div class="hint">WASD/setas mover · espaço atacar · E abrir baú</div>
    </div>
    <div id="controls">
      <div id="joy" class="joy"><div id="thumb" class="joy__thumb"></div></div>
      <div class="btns">
        <button id="open" class="btn">BAÚ</button>
        <button id="attack" class="btn btn--fire">ATIRAR</button>
      </div>
    </div>
  `;

  const controls = new Input(
    document.getElementById('joy')!,
    document.getElementById('thumb')!,
    document.getElementById('attack')!,
    document.getElementById('open')!,
  );

  new Phaser.Game({
    type: Phaser.AUTO,
    parent: 'game',
    backgroundColor: '#5e9b46',
    scale: { mode: Phaser.Scale.RESIZE, autoCenter: Phaser.Scale.CENTER_BOTH },
    render: { antialias: true },
    scene: new GameScene(controls),
  });
}

// Fluxo: lobby (criar/entrar/QR/ready/start) -> partida.
void runLobby(app).then(startGame);
