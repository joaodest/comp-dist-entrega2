// Renderizacao do estilo ".io cartoon top-down" em Phaser Graphics.
// drawTerrain: mapa estatico (grama, rio, arvores, pedras, flores) desenhado uma vez.
// drawWorld: entidades dinamicas (zona segura + tom externo, baus, jogadores e
// armas direcionais) do GameState, desenhadas a cada frame.
import Phaser from 'phaser';
import type { GameState } from './types';
import { PX, WORLD_PX, worldToPx } from './config';

type Heading = { x: number; y: number };

export const GRASS = 0x5e9b46;
const OUT = 0x15140f;
const C = {
  grassDark: 0x4f8a3b,
  grassLite: 0x6fb04f,
  sand: 0xd9c489,
  bank: 0x8d7d46,
  water: 0x3f80c4,
  shallow: 0x67a0d6,
  treeDark: 0x2f5a35,
  treeMid: 0x3f7a44,
  treeLite: 0x59b15a,
  bush: 0x46934a,
  rock: 0x98a1a4,
  rockHi: 0xc7cdd0,
  skin: 0xe7c08a,
  skinYou: 0xf3d089,
  shirt: 0x4b6cc4,
  shirtYou: 0x2fbf5e,
  dead: 0x7c7f88,
  crate: 0xb5702f,
  crateDark: 0x3a2f1d,
  crateLid: 0xd2944b,
  danger: 0xff5d6c,
  safeLine: 0x73d4ff,
  storm: 0x141a33,
  flowerA: 0xf5d76e,
  flowerB: 0xe7e2ef,
  flowerC: 0xe06aa6,
};

// Cor da arma equipada (cano), por tipo.
const WEAPON_COLOR: Record<string, number> = {
  pistol: 0x3a3f47,
  rifle: 0x2b3a2e,
  shotgun: 0x5a3a22,
};
// Comprimento/espessura do cano por arma (em px).
const WEAPON_GUN: Record<string, { len: number; w: number }> = {
  pistol: { len: 0.85 * PX, w: 5 },
  rifle: { len: 1.45 * PX, w: 4.5 },
  shotgun: { len: 1.1 * PX, w: 7 },
};

// Mapa decorativo (coordenadas de mundo -100..100). O backend nao tem terreno;
// estes elementos sao puramente visuais do cliente.
const TREES: [number, number, number][] = [
  [-66, 62, 5], [58, 66, 5], [-62, -58, 5], [70, -54, 5],
  [-82, 12, 4.2], [82, -18, 4.2], [26, 82, 4], [-30, -82, 4],
  [78, 38, 4.5], [-78, -26, 4.5], [42, -74, 4], [-46, 74, 4],
  [-20, 40, 4], [40, -40, 4.3], [-44, -44, 4], [18, -20, 3.8],
  [-90, -68, 4.2], [90, 70, 4.2], [4, -90, 4], [-8, 90, 4],
];
const ROCKS: [number, number, number][] = [
  [-18, 26, 2.2], [22, -14, 2.4], [2, -46, 2], [-50, -6, 1.8], [54, 22, 2.2], [14, 54, 1.8],
  [-60, 40, 2], [60, -50, 2.2], [-30, 70, 1.9], [70, 30, 2.1],
];
const BUSHES: [number, number, number][] = [
  [-36, 48, 1.6], [48, -44, 1.7], [-56, 28, 1.4], [32, 36, 1.5], [-24, -32, 1.6], [66, 8, 1.4], [-12, 68, 1.3], [18, -64, 1.5],
  [40, 60, 1.5], [-64, -40, 1.6], [-80, 56, 1.5], [80, -36, 1.5], [10, 28, 1.3], [-10, -10, 1.4],
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

function shadow(g: Phaser.GameObjects.Graphics, px: number, py: number, r: number) {
  g.fillStyle(0x0c1a0c, 0.18);
  g.fillEllipse(px, py + r * 0.5, r * 2.05, r * 1.15);
}

function tree(g: Phaser.GameObjects.Graphics, wx: number, wy: number, wr: number) {
  const p = worldToPx(wx, wy);
  const r = wr * PX;
  shadow(g, p.x + r * 0.12, p.y + r * 0.18, r);
  g.fillStyle(C.treeDark, 1);
  g.lineStyle(4, OUT, 1);
  const outer = blobPoints(p.x, p.y, r, 11, r * 0.08, 0);
  g.fillPoints(outer, true);
  g.strokePoints(outer, true);
  g.fillStyle(C.treeMid, 1);
  g.fillPoints(blobPoints(p.x, p.y, r * 0.7, 11, r * 0.06, 1.3), true);
  g.fillStyle(C.treeLite, 1);
  g.fillPoints(blobPoints(p.x - r * 0.12, p.y - r * 0.12, r * 0.38, 10, r * 0.05, 2.2), true);
}

function bush(g: Phaser.GameObjects.Graphics, wx: number, wy: number, wr: number) {
  const p = worldToPx(wx, wy);
  const r = wr * PX;
  shadow(g, p.x, p.y, r * 0.8);
  g.fillStyle(C.bush, 1);
  g.lineStyle(3, OUT, 0.9);
  const body = blobPoints(p.x, p.y, r, 7, r * 0.14, 0.4);
  g.fillPoints(body, true);
  g.strokePoints(body, true);
  g.fillStyle(C.treeLite, 0.8);
  g.fillPoints(blobPoints(p.x - r * 0.15, p.y - r * 0.12, r * 0.5, 7, r * 0.1, 1.7), true);
}

function rock(g: Phaser.GameObjects.Graphics, wx: number, wy: number, wr: number) {
  const p = worldToPx(wx, wy);
  const r = wr * PX;
  shadow(g, p.x, p.y, r);
  g.fillStyle(C.rock, 1);
  g.lineStyle(3.5, OUT, 1);
  const body = blobPoints(p.x, p.y, r, 5, r * 0.16, 0.6);
  g.fillPoints(body, true);
  g.strokePoints(body, true);
  g.fillStyle(C.rockHi, 0.9);
  g.fillPoints(blobPoints(p.x - r * 0.2, p.y - r * 0.18, r * 0.46, 5, r * 0.12, 1.4), true);
}

function flower(g: Phaser.GameObjects.Graphics, px: number, py: number, color: number) {
  g.fillStyle(color, 0.95);
  for (let i = 0; i < 5; i++) {
    const a = (i / 5) * Math.PI * 2;
    g.fillCircle(px + Math.cos(a) * 3, py + Math.sin(a) * 3, 2.1);
  }
  g.fillStyle(0xffe9a3, 1);
  g.fillCircle(px, py, 2);
}

export function drawTerrain(g: Phaser.GameObjects.Graphics) {
  // base de grama (mundo + margem)
  g.fillStyle(GRASS, 1);
  g.fillRect(-400, -400, WORLD_PX + 800, WORLD_PX + 800);

  // manchas de grama (claras e escuras) para dar textura
  for (const [wx, wy, wr, col, al] of [
    [-40, 36, 14, C.grassDark, 0.5], [44, -32, 13, C.grassDark, 0.5], [-12, -60, 13, C.grassLite, 0.4],
    [60, 56, 12, C.grassDark, 0.45], [-68, -40, 12, C.grassLite, 0.4], [36, 72, 12, C.grassDark, 0.4],
    [-80, 52, 13, C.grassLite, 0.35], [80, -64, 12, C.grassDark, 0.4],
    [-30, -20, 11, C.grassDark, 0.4], [24, 18, 10, C.grassLite, 0.35], [72, 8, 11, C.grassDark, 0.4],
  ] as const) {
    const p = worldToPx(wx, wy);
    g.fillStyle(col, al);
    g.fillPoints(blobPoints(p.x, p.y, wr * PX, 4, wr * PX * 0.25, wx), true);
  }

  // clareira de areia no centro (ponto de spawn)
  const center = worldToPx(0, 0);
  g.fillStyle(C.sand, 0.55);
  g.fillPoints(blobPoints(center.x, center.y, 7 * PX, 6, 7 * PX * 0.12, 0.5), true);
  g.fillStyle(C.grassLite, 0.25);
  g.fillCircle(center.x, center.y, 4.2 * PX);

  // rio (polilinha ondulada vertical)
  const river: Phaser.Math.Vector2[] = [];
  for (let wy = 108; wy >= -108; wy -= 4) {
    const wx = 10 * Math.sin(wy / 18);
    const p = worldToPx(wx, wy);
    river.push(new Phaser.Math.Vector2(p.x, p.y));
  }
  g.lineStyle(12 * PX, C.bank, 1);
  g.strokePoints(river, false);
  g.lineStyle(7 * PX, C.water, 1);
  g.strokePoints(river, false);
  g.lineStyle(3 * PX, C.shallow, 0.5);
  g.strokePoints(river, false);

  // flores espalhadas
  const palette = [C.flowerA, C.flowerB, C.flowerC];
  let seed = 1337;
  const rnd = () => {
    seed = (seed * 9301 + 49297) % 233280;
    return seed / 233280;
  };
  for (let i = 0; i < 220; i++) {
    const wx = (rnd() - 0.5) * 196;
    const wy = (rnd() - 0.5) * 196;
    if (Math.abs(10 * Math.sin(wy / 18) - wx) < 12) continue; // evita o rio
    const p = worldToPx(wx, wy);
    flower(g, p.x, p.y, palette[i % palette.length]);
  }

  for (const [x, y, r] of BUSHES) bush(g, x, y, r);
  for (const [x, y, r] of ROCKS) rock(g, x, y, r);
  for (const [x, y, r] of TREES) tree(g, x, y, r);
}

function drawDangerZone(g: Phaser.GameObjects.Graphics, cx: number, cy: number, r: number, t: number) {
  // Tom "tempestade" cobrindo tudo fora da zona: um anel grosso desenhado com
  // borda interna exatamente no raio seguro cobre toda a area externa visivel.
  const band = WORLD_PX * 2;
  g.lineStyle(band, C.storm, 0.42);
  g.strokeCircle(cx, cy, r + band / 2);

  // faixa avermelhada de perigo logo na borda externa
  g.lineStyle(10, C.danger, 0.22);
  g.strokeCircle(cx, cy, r + 6);

  // muralha da zona segura: glow + linha brilhante pulsante
  const pulse = 0.55 + 0.35 * Math.sin(t / 220);
  for (let i = 3; i >= 1; i--) {
    g.lineStyle(4 + i * 4, C.safeLine, 0.06 * pulse + 0.02);
    g.strokeCircle(cx, cy, r + i * 3);
  }
  g.lineStyle(3, C.safeLine, 0.95);
  g.strokeCircle(cx, cy, r);

  // marcadores tracejados girando na borda
  g.lineStyle(3, 0xffffff, 0.6);
  const ticks = 48;
  for (let i = 0; i < ticks; i++) {
    if (i % 2 === 0) continue;
    const a = (i / ticks) * Math.PI * 2 + t / 3000;
    const a2 = a + (Math.PI * 2) / ticks / 2;
    g.beginPath();
    g.arc(cx, cy, r, a, a2, false);
    g.strokePath();
  }
}

function drawChest(g: Phaser.GameObjects.Graphics, ch: GameState['chests'][number], t: number) {
  const p = worldToPx(ch.x, ch.y);
  const sz = 1.6 * PX;
  const half = sz / 2;
  shadow(g, p.x, p.y, half);
  if (ch.isOpened) {
    g.lineStyle(3, C.crateDark, 0.85);
    g.strokeRect(p.x - half, p.y - half * 0.4, sz, sz * 0.7);
    g.fillStyle(C.crateDark, 0.35);
    g.fillRect(p.x - half, p.y - half * 0.4, sz, sz * 0.7);
    return;
  }
  // brilho pulsante para atrair (bau fechado)
  const glow = 0.4 + 0.3 * Math.sin(t / 320);
  g.fillStyle(WEAPON_COLOR[ch.weapon] ?? 0xffe08a, 0.18 * glow);
  g.fillCircle(p.x, p.y, sz * 0.95);

  g.fillStyle(C.crate, 1);
  g.lineStyle(3, OUT, 1);
  g.fillRect(p.x - half, p.y - half, sz, sz);
  g.strokeRect(p.x - half, p.y - half, sz, sz);
  // tampa
  g.fillStyle(C.crateLid, 1);
  g.fillRect(p.x - half, p.y - half, sz, sz * 0.34);
  g.strokeRect(p.x - half, p.y - half, sz, sz * 0.34);
  // ferragens
  g.fillStyle(C.crateDark, 1);
  g.fillRect(p.x - 2, p.y - half, 4, sz);
  g.fillStyle(0xffe08a, 1);
  g.fillRect(p.x - 3, p.y - 3, 6, 6);
}

function healthBar(g: Phaser.GameObjects.Graphics, px: number, py: number, rad: number, hp: number) {
  const w = rad * 2.4;
  const h = 4.5;
  const x = px - w / 2;
  const y = py - rad - 11;
  const k = Math.max(0, Math.min(1, hp / 100));
  g.fillStyle(0x000000, 0.45);
  g.fillRoundedRect(x - 1, y - 1, w + 2, h + 2, 3);
  const col = k > 0.5 ? 0x2fbf5e : k > 0.25 ? 0xf5d76e : 0xff5d6c;
  g.fillStyle(col, 1);
  g.fillRoundedRect(x, y, Math.max(2, w * k), h, 2.5);
}

function drawPlayer(
  g: Phaser.GameObjects.Graphics,
  px: number,
  py: number,
  rad: number,
  alive: boolean,
  you: boolean,
  weapon: string,
  heading: Heading,
  hp: number,
) {
  shadow(g, px, py, rad);

  if (!alive) {
    g.fillStyle(C.dead, 0.85);
    g.lineStyle(3, OUT, 0.8);
    g.fillCircle(px, py, rad);
    g.strokeCircle(px, py, rad);
    // X marcando eliminado
    g.lineStyle(3, 0x3a3d44, 0.9);
    const d = rad * 0.45;
    g.lineBetween(px - d, py - d, px + d, py + d);
    g.lineBetween(px - d, py + d, px + d, py - d);
    return;
  }

  const hx = heading.x;
  const hy = heading.y;

  // arma (cano direcional) sai por tras do corpo apontando para o heading
  const gun = WEAPON_GUN[weapon] ?? WEAPON_GUN.pistol;
  const gx0 = px + hx * rad * 0.2;
  const gy0 = py + hy * rad * 0.2;
  const gx1 = px + hx * (rad + gun.len);
  const gy1 = py + hy * (rad + gun.len);
  g.lineStyle(gun.w + 2, OUT, 1);
  g.lineBetween(gx0, gy0, gx1, gy1);
  g.lineStyle(gun.w, WEAPON_COLOR[weapon] ?? 0x3a3f47, 1);
  g.lineBetween(gx0, gy0, gx1, gy1);
  // ponta do cano
  g.fillStyle(0x20242a, 1);
  g.fillCircle(gx1, gy1, gun.w * 0.6);

  // corpo (camisa) + cabeca (pele)
  g.fillStyle(you ? C.shirtYou : C.shirt, 1);
  g.lineStyle(3.5, OUT, 1);
  g.fillCircle(px, py, rad);
  g.strokeCircle(px, py, rad);
  g.fillStyle(you ? C.skinYou : C.skin, 1);
  g.fillCircle(px, py, rad * 0.62);

  // olhos voltados para a direcao
  const ex = px + hx * rad * 0.32;
  const ey = py + hy * rad * 0.32;
  const perpX = -hy;
  const perpY = hx;
  g.fillStyle(0x1a1c22, 1);
  g.fillCircle(ex + perpX * rad * 0.26, ey + perpY * rad * 0.26, 2.1);
  g.fillCircle(ex - perpX * rad * 0.26, ey - perpY * rad * 0.26, 2.1);

  if (you) {
    g.lineStyle(2.5, 0xffffff, 0.7);
    g.strokeCircle(px, py, rad + 4);
  }

  healthBar(g, px, py, rad, hp);
}

export function drawWorld(
  g: Phaser.GameObjects.Graphics,
  s: GameState,
  render: Map<string, { x: number; y: number }>,
  headings: Map<string, Heading>,
  myId: string,
) {
  const t = performance.now();
  const c = worldToPx(s.safeZone.centerX, s.safeZone.centerY);
  const r = s.safeZone.radius * PX;

  drawDangerZone(g, c.x, c.y, r, t);

  for (const ch of s.chests) drawChest(g, ch, t);

  // jogadores: mortos primeiro, "voce" por ultimo
  const ordered = [
    ...s.players.filter((p) => !p.isAlive),
    ...s.players.filter((p) => p.isAlive && p.playerId !== myId),
    ...s.players.filter((p) => p.playerId === myId),
  ];
  for (const p of ordered) {
    const rp = render.get(p.playerId) ?? worldToPx(p.x, p.y);
    const you = p.playerId === myId;
    const rad = (you ? 1.05 : 0.92) * PX;
    const inZone = Math.hypot(p.x - s.safeZone.centerX, p.y - s.safeZone.centerY) <= s.safeZone.radius;
    const heading = headings.get(p.playerId) ?? { x: 1, y: 0 };

    if (p.isAlive && !inZone) {
      const pulse = 0.5 + 0.4 * Math.sin(t / 160);
      g.lineStyle(3, C.danger, 0.5 + 0.4 * pulse);
      g.strokeCircle(rp.x, rp.y, rad + 5);
    }

    drawPlayer(g, rp.x, rp.y, rad, p.isAlive, you, p.weapon, heading, p.health);
  }
}
