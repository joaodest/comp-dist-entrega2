// Renderizacao do estilo ".io cartoon top-down" em Phaser Graphics.
// drawTerrain: mapa estatico (grama, rio, arvores, pedras) desenhado uma vez.
// drawWorld: entidades dinamicas (zona, baus, jogadores) do GameState, por frame.
import Phaser from 'phaser';
import type { GameState } from './types';
import { PX, WORLD_PX, worldToPx } from './config';

export const GRASS = 0x5e9b46;
const OUT = 0x15140f;
const C = {
  grassDark: 0x4f8a3b,
  bank: 0x8d7d46,
  water: 0x3f80c4,
  shallow: 0x67a0d6,
  treeDark: 0x34623a,
  treeMid: 0x3f7a44,
  treeLite: 0x59b15a,
  rock: 0x98a1a4,
  rockHi: 0xc7cdd0,
  skin: 0xe7c08a,
  skinYou: 0xf3d089,
  dead: 0x7c7f88,
  gun: 0x34525c,
  crate: 0xb5702f,
  crateDark: 0x3a2f1d,
  plank: 0x5a4a2e,
  danger: 0xff5d6c,
  safeLine: 0x73d4ff,
};

// Mapa decorativo (coordenadas de mundo -50..50). O backend nao tem terreno;
// estes elementos sao puramente visuais do cliente.
const TREES: [number, number, number][] = [
  [-33, 31, 5], [29, 33, 5], [-31, -29, 5], [35, -27, 5],
  [-41, 6, 4.2], [41, -9, 4.2], [13, 41, 4], [-15, -41, 4],
  [39, 19, 4.5], [-39, -13, 4.5], [21, -37, 4], [-23, 37, 4],
];
const ROCKS: [number, number, number][] = [
  [-9, 13, 2.2], [11, -7, 2.4], [1, -23, 2], [-25, -3, 1.8], [27, 11, 2.2], [7, 27, 1.8],
];

function blobPoints(cx: number, cy: number, r: number, bumps: number, amp: number, phase = 0): Phaser.Math.Vector2[] {
  const n = Math.max(28, bumps * 6);
  const pts: Phaser.Math.Vector2[] = [];
  for (let i = 0; i < n; i++) {
    const a = (i / n) * Math.PI * 2;
    const rr = r + Math.sin(a * bumps + phase) * amp;
    pts.push(new Phaser.Math.Vector2(cx + Math.cos(a) * rr, cy + Math.sin(a) * rr));
  }
  return pts;
}

function tree(g: Phaser.GameObjects.Graphics, wx: number, wy: number, wr: number) {
  const p = worldToPx(wx, wy);
  const r = wr * PX;
  g.fillStyle(C.treeDark, 1);
  g.lineStyle(4, OUT, 1);
  const outer = blobPoints(p.x, p.y, r, 11, r * 0.08, 0);
  g.fillPoints(outer, true);
  g.strokePoints(outer, true);
  g.fillStyle(C.treeMid, 1);
  g.fillPoints(blobPoints(p.x, p.y, r * 0.7, 11, r * 0.06, 1.3), true);
  g.fillStyle(C.treeLite, 1);
  g.fillPoints(blobPoints(p.x, p.y, r * 0.38, 10, r * 0.05, 2.2), true);
}

function rock(g: Phaser.GameObjects.Graphics, wx: number, wy: number, wr: number) {
  const p = worldToPx(wx, wy);
  const r = wr * PX;
  g.fillStyle(C.rock, 1);
  g.lineStyle(3.5, OUT, 1);
  const body = blobPoints(p.x, p.y, r, 5, r * 0.16, 0.6);
  g.fillPoints(body, true);
  g.strokePoints(body, true);
  g.fillStyle(C.rockHi, 0.9);
  g.fillPoints(blobPoints(p.x - r * 0.2, p.y - r * 0.18, r * 0.46, 5, r * 0.12, 1.4), true);
}

export function drawTerrain(g: Phaser.GameObjects.Graphics) {
  // base de grama (mundo + margem)
  g.fillStyle(GRASS, 1);
  g.fillRect(-400, -400, WORLD_PX + 800, WORLD_PX + 800);

  // manchas de grama
  g.fillStyle(C.grassDark, 0.5);
  for (const [wx, wy, wr] of [[-20, 18, 8], [22, -16, 7], [-6, -30, 7], [30, 28, 6], [-34, -20, 6]] as const) {
    const p = worldToPx(wx, wy);
    g.fillPoints(blobPoints(p.x, p.y, wr * PX, 4, wr * PX * 0.25, wx), true);
  }

  // rio (polilinha ondulada vertical)
  const river: Phaser.Math.Vector2[] = [];
  for (let wy = 54; wy >= -54; wy -= 4) {
    const wx = 6 * Math.sin(wy / 13);
    const p = worldToPx(wx, wy);
    river.push(new Phaser.Math.Vector2(p.x, p.y));
  }
  g.lineStyle(12 * PX, C.bank, 1);
  g.strokePoints(river, false);
  g.lineStyle(7 * PX, C.water, 1);
  g.strokePoints(river, false);
  g.lineStyle(3 * PX, C.shallow, 0.5);
  g.strokePoints(river, false);

  for (const [x, y, r] of ROCKS) rock(g, x, y, r);
  for (const [x, y, r] of TREES) tree(g, x, y, r);
}

export function drawWorld(
  g: Phaser.GameObjects.Graphics,
  s: GameState,
  render: Map<string, { x: number; y: number }>,
  myId: string,
) {
  const c = worldToPx(0, 0);
  const r = s.safeZone.radius * PX;

  // muralha da zona segura: glow externo + linha brilhante
  for (let i = 3; i >= 1; i--) {
    g.lineStyle(4 + i * 4, C.danger, 0.06);
    g.strokeCircle(c.x, c.y, r + i * 3);
  }
  g.lineStyle(3, C.safeLine, 0.95);
  g.strokeCircle(c.x, c.y, r);

  // baus
  for (const ch of s.chests) {
    const p = worldToPx(ch.x, ch.y);
    const sz = 1.5 * PX;
    if (ch.isOpened) {
      g.lineStyle(3, C.crate, 0.8);
      g.strokeRect(p.x - sz / 2, p.y - sz / 2, sz, sz);
    } else {
      g.fillStyle(C.crate, 1);
      g.fillRect(p.x - sz / 2, p.y - sz / 2, sz, sz);
      g.fillStyle(C.crateDark, 1);
      g.fillRect(p.x - sz / 2 + 3, p.y - sz / 2 + 3, sz - 6, sz - 6);
      g.lineStyle(3, OUT, 1);
      g.strokeRect(p.x - sz / 2, p.y - sz / 2, sz, sz);
    }
  }

  // jogadores: mortos primeiro, "voce" por ultimo
  const ordered = [
    ...s.players.filter((p) => !p.isAlive),
    ...s.players.filter((p) => p.isAlive && p.playerId !== myId),
    ...s.players.filter((p) => p.playerId === myId),
  ];
  for (const p of ordered) {
    const rp = render.get(p.playerId) ?? worldToPx(p.x, p.y);
    const you = p.playerId === myId;
    const rad = (you ? 1.05 : 0.9) * PX;
    const inZone = Math.hypot(p.x, p.y) <= s.safeZone.radius;

    if (p.isAlive && !inZone) {
      g.lineStyle(3, C.danger, 0.9);
      g.strokeCircle(rp.x, rp.y, rad + 5);
    }
    if (p.isAlive) {
      g.fillStyle(C.gun, 1);
      g.lineStyle(2, OUT, 1);
      const gw = 0.85 * PX;
      const gh = 6;
      g.fillRect(rp.x + rad - 3, rp.y - gh / 2, gw, gh);
      g.strokeRect(rp.x + rad - 3, rp.y - gh / 2, gw, gh);
    }
    g.fillStyle(p.isAlive ? (you ? C.skinYou : C.skin) : C.dead, 1);
    g.lineStyle(3.5, OUT, 1);
    g.fillCircle(rp.x, rp.y, rad);
    g.strokeCircle(rp.x, rp.y, rad);
    if (you) {
      g.lineStyle(2.5, 0xffffff, 0.55);
      g.strokeCircle(rp.x, rp.y, rad + 4);
    }
  }
}
