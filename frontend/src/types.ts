// Espelha os contratos de proto/match/v1/match.proto na serializacao JSON do
// grpc-gateway: campos em camelCase e inteiros int64 como string ("tick":"1").

export interface PlayerSnapshot {
  playerId: string;
  x: number;
  y: number;
  isAlive: boolean;
  health: number;
  weapon: string;
  eliminations: number;
  damageDealt: number;
  damageTaken: number;
  survivedTicks: string;
}

export interface ChestSnapshot {
  chestId: string;
  x: number;
  y: number;
  isOpened: boolean;
  weapon: string;
  openedByPlayerId: string;
}

export interface SafeZoneSnapshot {
  centerX: number;
  centerY: number;
  radius: number;
  phase: string;
}

export interface RankingEntry {
  playerId: string;
  place: number;
  isAlive: boolean;
  health: number;
  eliminations: number;
  damageDealt: number;
  survivedTicks: string;
}

export interface GameState {
  tick: string;
  players: PlayerSnapshot[];
  chests: ChestSnapshot[];
  safeZone: SafeZoneSnapshot;
  ranking: RankingEntry[];
  matchEnded: boolean;
  remainingTicks: string;
}

/** Input enviado a POST /v1/match/stream (PlayerInput do match.proto). */
export interface PlayerInput {
  playerId: string;
  moveX: number;
  moveY: number;
  isAttacking: boolean;
  inputSequence: number;
  openChest: boolean;
  targetPlayerId?: string;
  aimX: number;
  aimY: number;
  // roomId roteia o input para a partida da sala (vazio = partida global/demo).
  roomId?: string;
}
