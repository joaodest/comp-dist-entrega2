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

// --- Cliente ---
export const PX = 16; // pixels por unidade de mundo
export const WORLD_PX = ARENA_HALF * 2 * PX; // 3840
// id unico por sessao: evita colisao de identidade e input "stale" (o servidor
// guarda o ultimo inputSequence por jogador; reusar o mesmo id trava o input).
export const MY_ID = 'web-' + Math.random().toString(36).slice(2, 8);
export const SEND_MS = 90; // intervalo de envio de input ao Gateway
export const GATEWAY = '/v1/match/stream'; // via proxy do Vite -> :8080 (legado)

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

export function safeZoneRadiusAtTick(tick: number): number {
  const p = Math.min(Math.max(tick / MAX_MATCH_TICKS, 0), 1);
  return SAFE_ZONE_INITIAL - (SAFE_ZONE_INITIAL - SAFE_ZONE_FINAL) * p;
}

export function phaseAtTick(tick: number): number {
  return Math.min(PHASES - 1, Math.floor(tick / (MAX_MATCH_TICKS / PHASES)));
}
