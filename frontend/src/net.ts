// Driver de partida. LiveDriver fala com o Gateway real (POST /v1/match/stream).
// OfflineDriver simula localmente para o cliente rodar sem backend (fallback).
import type { GameState, PlayerInput, PlayerSnapshot } from './types';
import { GATEWAY, ARENA_HALF } from './config';
import { session } from './session';
import { buildSnapshot } from './mock';

export type Mode = 'live' | 'offline';

export interface Driver {
  readonly mode: Mode;
  step(input: PlayerInput): Promise<GameState>;
}

export class LiveDriver implements Driver {
  readonly mode: Mode = 'live';
  async step(input: PlayerInput): Promise<GameState> {
    const res = await fetch(GATEWAY, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(input),
    });
    if (!res.ok) throw new Error(`gateway HTTP ${res.status}`);
    return (await res.json()) as GameState;
  }
}

function clamp(v: number, a: number, b: number): number {
  return v < a ? a : v > b ? b : v;
}

export class OfflineDriver implements Driver {
  readonly mode: Mode = 'offline';
  private tick = 0;
  private px: number;
  private py: number;

  constructor(start?: { x: number; y: number }) {
    this.px = start?.x ?? 0;
    this.py = start?.y ?? 0;
  }

  async step(input: PlayerInput): Promise<GameState> {
    this.tick++;
    const mag = Math.hypot(input.moveX, input.moveY);
    if (mag > 0.01) {
      const stepLen = Math.min(2.5, mag * 2.5);
      this.px = clamp(this.px + (input.moveX / mag) * stepLen, -ARENA_HALF, ARENA_HALF);
      this.py = clamp(this.py + (input.moveY / mag) * stepLen, -ARENA_HALF, ARENA_HALF);
    }
    const base = buildSnapshot(this.tick);
    const me: PlayerSnapshot = {
      playerId: session.myId,
      x: this.px,
      y: this.py,
      isAlive: true,
      health: 100,
      weapon: 'rifle',
      eliminations: 0,
      damageDealt: 0,
      damageTaken: 0,
      survivedTicks: String(this.tick),
    };
    return { ...base, players: [...base.players, me] };
  }
}
