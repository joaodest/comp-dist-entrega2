// Transporte da partida.
// RealtimeClient (Fase 4): WebSocket persistente com o Gateway. Envia inputs
// sequenciados e recebe snapshots publicados pelo relogio do servidor.
// OfflineDriver: fallback local (mock) quando o backend nao esta acessivel.
import type { GameState, PlayerInput, PlayerSnapshot } from './types';
import { ARENA_HALF, matchWsUrl, moveWithRockCollision, SEND_MS, SERVER_TICK_HZ } from './config';
import { session } from './session';
import { buildSnapshot, rankPlayers } from './mock';

export type Mode = 'live' | 'offline';
export type RealtimeStatus = 'connecting' | 'live' | 'offline';

/** Cliente WebSocket de tempo real para uma sessao de partida. */
export class RealtimeClient {
  private ws: WebSocket | null = null;
  private latest: GameState | null = null;
  private status: RealtimeStatus = 'connecting';
  private everConnected = false;
  private reconnected = false;
  private closed = false;
  private connectTimer: number | undefined;

  constructor(
    private readonly roomId: string,
    private readonly playerId: string,
    private readonly onStatus: (status: RealtimeStatus) => void,
  ) {}

  connect(): void {
    this.setStatus('connecting');
    let ws: WebSocket;
    try {
      ws = new WebSocket(matchWsUrl(this.roomId, this.playerId));
    } catch {
      this.setStatus('offline');
      return;
    }
    this.ws = ws;

    // Se nao abrir em tempo habil, assume backend indisponivel -> offline.
    this.connectTimer = window.setTimeout(() => {
      if (!this.everConnected) {
        try {
          ws.close();
        } catch {
          /* ignore */
        }
        this.setStatus('offline');
      }
    }, 3000);

    ws.onopen = () => {
      this.everConnected = true;
      window.clearTimeout(this.connectTimer);
      this.setStatus('live');
    };

    ws.onmessage = (event) => {
      try {
        this.latest = JSON.parse(event.data as string) as GameState;
      } catch {
        /* ignora frames invalidos */
      }
    };

    ws.onclose = () => {
      window.clearTimeout(this.connectTimer);
      if (this.closed) return;
      // Uma tentativa de reconexao se a conexao ja estava ativa; senao, offline.
      if (this.everConnected && !this.reconnected) {
        this.reconnected = true;
        this.everConnected = false;
        this.setStatus('connecting');
        window.setTimeout(() => this.connect(), 1000);
        return;
      }
      this.setStatus('offline');
    };

    ws.onerror = () => {
      // onclose cuida da transicao de estado.
    };
  }

  sendInput(input: PlayerInput): void {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(input));
    }
  }

  getState(): GameState | null {
    return this.latest;
  }

  getStatus(): RealtimeStatus {
    return this.status;
  }

  close(): void {
    this.closed = true;
    window.clearTimeout(this.connectTimer);
    this.ws?.close();
  }

  private setStatus(status: RealtimeStatus): void {
    if (this.status === status) return;
    this.status = status;
    this.onStatus(status);
  }
}

function clamp(v: number, a: number, b: number): number {
  return v < a ? a : v > b ? b : v;
}

/** Simulacao local de fallback (sem backend). */
export class OfflineDriver {
  readonly mode: Mode = 'offline';
  private tick = 0;
  private tickCarry = 0;
  private px: number;
  private py: number;

  constructor(start?: { x: number; y: number }) {
    this.px = start?.x ?? 0;
    this.py = start?.y ?? 0;
  }

  step(input: PlayerInput): GameState {
    this.tickCarry += (SEND_MS / 1000) * SERVER_TICK_HZ;
    const elapsedTicks = Math.max(1, Math.floor(this.tickCarry));
    this.tickCarry -= elapsedTicks;
    this.tick += elapsedTicks;
    const mag = Math.hypot(input.moveX, input.moveY);
    if (mag > 0.01) {
      const stepLen = Math.min(2.5, mag * 2.5);
      const next = moveWithRockCollision(
        { x: this.px, y: this.py },
        {
          x: clamp(this.px + (input.moveX / mag) * stepLen, -ARENA_HALF, ARENA_HALF),
          y: clamp(this.py + (input.moveY / mag) * stepLen, -ARENA_HALF, ARENA_HALF),
        },
      );
      this.px = next.x;
      this.py = next.y;
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
    const players = [...base.players, me];
    return { ...base, players, ranking: rankPlayers(players) };
  }
}
