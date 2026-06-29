// Constantes da arena espelhadas de internal/game/server.go (fonte de verdade).
export const ARENA_HALF = 120;
export const SERVER_TICK_HZ = 15;
export const MATCH_SECONDS = 5 * 60;
export const MAX_MATCH_TICKS = MATCH_SECONDS * SERVER_TICK_HZ;
export const SAFE_ZONE_INITIAL = 108;
export const SAFE_ZONE_FINAL = 10;
export const PHASES = 5;

export const WEAPONS = ['pistol', 'rifle', 'shotgun'] as const;
export type Weapon = (typeof WEAPONS)[number];
export const WEAPON_RANGE_UNITS: Record<Weapon, number> = {
  pistol: 10,
  rifle: 16,
  shotgun: 8,
};

// --- Cliente ---
export const PX = 16; // pixels por unidade de mundo
export const WORLD_PX = ARENA_HALF * 2 * PX; // 3840
// id unico por sessao: evita colisao de identidade e input "stale" (o servidor
// guarda o ultimo inputSequence por jogador; reusar o mesmo id trava o input).
export const MY_ID = 'web-' + Math.random().toString(36).slice(2, 8);
export const SEND_MS = 90; // intervalo de envio de input ao Gateway
export const GATEWAY = '/v1/match/stream'; // via proxy do Vite -> :8080 (legado)
export const PLAYER_COLLISION_RADIUS = 1.05;

export interface WorldPoint {
  x: number;
  y: number;
}

export interface RockObstacle {
  wx: number;
  wy: number;
  wr: number;
}

export const ROCK_OBSTACLES: RockObstacle[] = [
  { wx: -18, wy: 26, wr: 2.2 },
  { wx: 22, wy: -14, wr: 2.4 },
  { wx: 2, wy: -46, wr: 2 },
  { wx: -50, wy: -6, wr: 1.8 },
  { wx: 54, wy: 22, wr: 2.2 },
  { wx: 14, wy: 54, wr: 1.8 },
  { wx: -60, wy: 40, wr: 2 },
  { wx: 60, wy: -50, wr: 2.2 },
  { wx: -30, wy: 70, wr: 1.9 },
  { wx: 70, wy: 30, wr: 2.1 },
  { wx: -96, wy: 70, wr: 2 },
  { wx: 94, wy: -72, wr: 2.1 },
  { wx: 78, wy: 92, wr: 1.9 },
  { wx: -82, wy: -94, wr: 2.1 },
  { wx: 104, wy: 46, wr: 1.8 },
  { wx: -106, wy: -42, wr: 1.9 },
];

/** URL do WebSocket de tempo real (Fase 4): /v1/match/ws via proxy -> :8080. */
export function matchWsUrl(roomId: string, playerId: string): string {
  const scheme = location.protocol === 'https:' ? 'wss' : 'ws';
  const params = new URLSearchParams({ room: roomId, player: playerId });
  return `${scheme}://${location.host}/v1/match/ws?${params.toString()}`;
}

/** Mundo (-ARENA_HALF..ARENA_HALF, +y para cima) -> pixels do canvas. */
export function worldToPx(x: number, y: number): { x: number; y: number } {
  return { x: (x + ARENA_HALF) * PX, y: (ARENA_HALF - y) * PX };
}

export function weaponRangeUnits(weapon: string | undefined): number {
  return WEAPON_RANGE_UNITS[(weapon ?? '') as Weapon] ?? WEAPON_RANGE_UNITS.pistol;
}

export function hasRockCollision(pos: WorldPoint, actorRadius = PLAYER_COLLISION_RADIUS): boolean {
  return ROCK_OBSTACLES.some((rock) => {
    const dx = pos.x - rock.wx;
    const dy = pos.y - rock.wy;
    const r = rock.wr + actorRadius;
    return dx * dx + dy * dy < r * r;
  });
}

export function rockBlocksLine(from: WorldPoint, to: WorldPoint, padding = 0): boolean {
  return firstRockHitT(from, to, padding) !== null;
}

export function firstRockHit(from: WorldPoint, to: WorldPoint, padding = 0): WorldPoint | null {
  const t = firstRockHitT(from, to, padding);
  if (t === null) return null;
  return {
    x: from.x + (to.x - from.x) * t,
    y: from.y + (to.y - from.y) * t,
  };
}

export function moveWithRockCollision(
  from: WorldPoint,
  to: WorldPoint,
  actorRadius = PLAYER_COLLISION_RADIUS,
): WorldPoint {
  if (rockPathClear(from, to, actorRadius)) return to;

  const xThenY = slideWithRocks(from, to, actorRadius, true);
  const yThenX = slideWithRocks(from, to, actorRadius, false);
  if (distanceSq(from, yThenX) > distanceSq(from, xThenY)) return yThenX;
  if (distanceSq(from, xThenY) > 0) return xThenY;

  const t = firstRockHitT(from, to, actorRadius);
  if (t === null) return from;
  const safeT = Math.max(0, t - 0.02);
  return {
    x: from.x + (to.x - from.x) * safeT,
    y: from.y + (to.y - from.y) * safeT,
  };
}

function slideWithRocks(from: WorldPoint, to: WorldPoint, actorRadius: number, xFirst: boolean): WorldPoint {
  let pos = from;
  const first = xFirst ? { x: to.x, y: from.y } : { x: from.x, y: to.y };
  if (rockPathClear(pos, first, actorRadius)) pos = first;

  const second = xFirst ? { x: pos.x, y: to.y } : { x: to.x, y: pos.y };
  if (rockPathClear(pos, second, actorRadius)) pos = second;
  return pos;
}

function rockPathClear(from: WorldPoint, to: WorldPoint, actorRadius: number): boolean {
  return !hasRockCollision(to, actorRadius) && firstRockHitT(from, to, actorRadius) === null;
}

function firstRockHitT(from: WorldPoint, to: WorldPoint, padding: number): number | null {
  let best: number | null = null;
  for (const rock of ROCK_OBSTACLES) {
    const t = segmentCircleHitT(from, to, { x: rock.wx, y: rock.wy }, rock.wr + padding);
    if (t !== null && (best === null || t < best)) best = t;
  }
  return best;
}

function segmentCircleHitT(from: WorldPoint, to: WorldPoint, center: WorldPoint, radius: number): number | null {
  const dx = to.x - from.x;
  const dy = to.y - from.y;
  const a = dx * dx + dy * dy;
  if (a <= 0) return null;

  const fx = from.x - center.x;
  const fy = from.y - center.y;
  const c = fx * fx + fy * fy - radius * radius;
  if (c <= 0) return 0;

  const b = 2 * (fx * dx + fy * dy);
  const disc = b * b - 4 * a * c;
  if (disc < 0) return null;

  const root = Math.sqrt(disc);
  const t1 = (-b - root) / (2 * a);
  if (t1 >= 0 && t1 <= 1) return t1;
  const t2 = (-b + root) / (2 * a);
  if (t2 >= 0 && t2 <= 1) return t2;
  return null;
}

function distanceSq(a: WorldPoint, b: WorldPoint): number {
  const dx = a.x - b.x;
  const dy = a.y - b.y;
  return dx * dx + dy * dy;
}

export function safeZoneRadiusAtTick(tick: number): number {
  const p = Math.min(Math.max(tick / MAX_MATCH_TICKS, 0), 1);
  return SAFE_ZONE_INITIAL - (SAFE_ZONE_INITIAL - SAFE_ZONE_FINAL) * p;
}

export function phaseAtTick(tick: number): number {
  return Math.min(PHASES - 1, Math.floor(tick / (MAX_MATCH_TICKS / PHASES)));
}
