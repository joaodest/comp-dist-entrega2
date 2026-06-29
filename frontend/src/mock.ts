// Snapshot mock fiel ao formato GameState. Os baús usam as posições/armas
// reais de internal/game/server.go (chestTemplates). A zona segura é recalculada
// a partir do tick (mesma fórmula do servidor) para o slider mostrar o
// encolhimento. Jogadores são um cenário representativo de meia-partida.

import type { GameState, PlayerSnapshot, ChestSnapshot, RankingEntry } from './types';
import { phaseAtTick, safeZoneRadiusAtTick, MAX_MATCH_TICKS } from './config';

const CHESTS: ChestSnapshot[] = [
  { chestId: 'chest-01', x: 3, y: 0, weapon: 'rifle', isOpened: true, openedByPlayerId: 'player-1' },
  { chestId: 'chest-02', x: -3, y: 0, weapon: 'shotgun', isOpened: false, openedByPlayerId: '' },
  { chestId: 'chest-03', x: 0, y: 3, weapon: 'pistol', isOpened: false, openedByPlayerId: '' },
  { chestId: 'chest-04', x: 28, y: 28, weapon: 'rifle', isOpened: true, openedByPlayerId: 'player-3' },
  { chestId: 'chest-05', x: -28, y: 28, weapon: 'shotgun', isOpened: false, openedByPlayerId: '' },
  { chestId: 'chest-06', x: 28, y: -28, weapon: 'pistol', isOpened: false, openedByPlayerId: '' },
  { chestId: 'chest-07', x: -28, y: -28, weapon: 'rifle', isOpened: false, openedByPlayerId: '' },
  { chestId: 'chest-08', x: 50, y: 0, weapon: 'shotgun', isOpened: false, openedByPlayerId: '' },
  { chestId: 'chest-09', x: 0, y: -50, weapon: 'pistol', isOpened: false, openedByPlayerId: '' },
  { chestId: 'chest-10', x: 72, y: 48, weapon: 'rifle', isOpened: false, openedByPlayerId: '' },
  { chestId: 'chest-11', x: -72, y: 48, weapon: 'pistol', isOpened: false, openedByPlayerId: '' },
  { chestId: 'chest-12', x: 72, y: -48, weapon: 'shotgun', isOpened: false, openedByPlayerId: '' },
  { chestId: 'chest-13', x: -72, y: -48, weapon: 'rifle', isOpened: false, openedByPlayerId: '' },
  { chestId: 'chest-14', x: 0, y: 84, weapon: 'shotgun', isOpened: false, openedByPlayerId: '' },
  { chestId: 'chest-15', x: 84, y: 0, weapon: 'pistol', isOpened: false, openedByPlayerId: '' },
  { chestId: 'chest-16', x: 0, y: -84, weapon: 'rifle', isOpened: false, openedByPlayerId: '' },
  { chestId: 'chest-17', x: -84, y: 0, weapon: 'shotgun', isOpened: false, openedByPlayerId: '' },
];

type Seed = { id: string; x: number; y: number; hp: number; w: string; alive: boolean; elim: number };

const SEEDS: Seed[] = [
  { id: 'player-1', x: 10, y: -14, hp: 78, w: 'rifle', alive: true, elim: 2 },
  { id: 'player-2', x: -64, y: 8, hp: 100, w: 'pistol', alive: true, elim: 0 },
  { id: 'player-3', x: 74, y: 42, hp: 64, w: 'shotgun', alive: true, elim: 1 },
  { id: 'player-4', x: -74, y: -48, hp: 42, w: 'rifle', alive: true, elim: 0 },
  { id: 'player-5', x: 18, y: 78, hp: 100, w: 'pistol', alive: true, elim: 0 },
  { id: 'player-6', x: -18, y: -72, hp: 53, w: 'shotgun', alive: true, elim: 3 },
  { id: 'player-7', x: 92, y: -10, hp: 18, w: 'rifle', alive: true, elim: 0 },
  { id: 'player-8', x: -88, y: 44, hp: 87, w: 'pistol', alive: true, elim: 1 },
  { id: 'player-9', x: 48, y: -70, hp: 0, w: 'rifle', alive: false, elim: 0 },
  { id: 'player-10', x: -92, y: 4, hp: 0, w: 'shotgun', alive: false, elim: 0 },
  { id: 'player-11', x: 34, y: 28, hp: 96, w: 'pistol', alive: true, elim: 0 },
  { id: 'player-12', x: -28, y: 88, hp: 71, w: 'rifle', alive: true, elim: 2 },
  { id: 'player-13', x: 82, y: 82, hp: 0, w: 'pistol', alive: false, elim: 0 },
  { id: 'player-14', x: -54, y: -18, hp: 35, w: 'shotgun', alive: true, elim: 0 },
];

export function buildSnapshot(tick: number): GameState {
  const players: PlayerSnapshot[] = SEEDS.map((s) => ({
    playerId: s.id,
    x: s.x,
    y: s.y,
    isAlive: s.alive,
    health: s.hp,
    weapon: s.w,
    eliminations: s.elim,
    damageDealt: s.elim * 100 + s.hp,
    damageTaken: 100 - s.hp,
    survivedTicks: String(s.alive ? tick : Math.floor(tick * 0.6)),
  }));

  return {
    tick: String(tick),
    players,
    chests: CHESTS,
    safeZone: {
      centerX: 0,
      centerY: 0,
      radius: safeZoneRadiusAtTick(tick),
      phase: String(phaseAtTick(tick)),
    },
    ranking: rankPlayers(players),
    matchEnded: tick >= MAX_MATCH_TICKS,
    remainingTicks: String(Math.max(0, MAX_MATCH_TICKS - tick)),
  };
}

export function rankPlayers(players: PlayerSnapshot[]): RankingEntry[] {
  return [...players]
    .sort((a, b) => {
      if (a.isAlive !== b.isAlive) return a.isAlive ? -1 : 1;
      if (a.isAlive && a.eliminations !== b.eliminations) return b.eliminations - a.eliminations;
      if (a.isAlive && a.health !== b.health) return b.health - a.health;
      if (Number(a.survivedTicks) !== Number(b.survivedTicks)) {
        return Number(b.survivedTicks) - Number(a.survivedTicks);
      }
      if (a.damageDealt !== b.damageDealt) return b.damageDealt - a.damageDealt;
      return a.playerId.localeCompare(b.playerId);
    })
    .map((p, i) => ({
      playerId: p.playerId,
      place: i + 1,
      isAlive: p.isAlive,
      health: p.health,
      eliminations: p.eliminations,
      damageDealt: p.damageDealt,
      survivedTicks: p.survivedTicks,
    }));
}
