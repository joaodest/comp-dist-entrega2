import Phaser from 'phaser';
import type { GameState, PlayerInput } from './types';
import { Input } from './input';
import type { RealtimeStatus } from './net';
import { OfflineDriver, RealtimeClient } from './net';
import { drawTerrain, drawWorld, GRASS } from './ioRender';
import { Bullets } from './bullets';
import { SEND_MS, worldToPx, PHASES, MAX_MATCH_TICKS, SERVER_TICK_HZ } from './config';
import { session } from './session';

type Heading = { x: number; y: number };

export class GameScene extends Phaser.Scene {
  private controls: Input;
  private rt!: RealtimeClient;
  private offline: OfflineDriver | null = null;
  private state: GameState | null = null;
  private renderPos = new Map<string, { x: number; y: number }>();
  private headings = new Map<string, Heading>();
  private terrain!: Phaser.GameObjects.Graphics;
  private dyn!: Phaser.GameObjects.Graphics;
  private fx!: Phaser.GameObjects.Graphics;
  private bullets = new Bullets();
  private follow!: Phaser.GameObjects.Arc;
  private accum = 0;
  private endShown = false;
  private lastFxTick = '';

  constructor(controls: Input) {
    super('game');
    this.controls = controls;
  }

  create() {
    this.cameras.main.setBackgroundColor(GRASS);
    this.terrain = this.add.graphics();
    drawTerrain(this.terrain);
    this.dyn = this.add.graphics();
    this.fx = this.add.graphics();

    const center = worldToPx(0, 0);
    this.follow = this.add.circle(center.x, center.y, 1, 0x000000, 0);
    this.cameras.main.setZoom(1.3);
    this.cameras.main.startFollow(this.follow, true, 0.15, 0.15);

    // Conexao WebSocket de tempo real com o Gateway (Fase 4). Em caso de falha,
    // o cliente cai para a simulacao local (OfflineDriver).
    this.rt = new RealtimeClient(session.roomId, session.myId, (status) => this.onStatus(status));
    this.rt.connect();
  }

  update(_time: number, delta: number) {
    this.accum += delta;
    if (this.accum >= SEND_MS) {
      this.accum = 0;
      if (!this.state?.matchEnded) {
        const input = this.withAutoTarget(this.controls.sample());
        if (input.isAttacking) this.tryLocalFire(input);
        if (this.offline) {
          this.state = this.offline.step(input);
        } else {
          this.rt.sendInput(input);
        }
      }
    }
    if (!this.offline) {
      const live = this.rt.getState();
      if (live) this.state = live;
    }
    if (this.state) this.draw();

    this.bullets.update(delta);
    this.bullets.draw(this.fx);
  }

  private onStatus(status: RealtimeStatus) {
    if (status === 'offline' && !this.offline) {
      this.offline = new OfflineDriver(this.currentPlayerWorld());
    }
    this.setStatus(status);
  }

  private draw() {
    const s = this.state!;
    for (const p of s.players) {
      const target = worldToPx(p.x, p.y);
      const cur = this.renderPos.get(p.playerId);
      if (!cur) this.renderPos.set(p.playerId, target);
      else {
        const dx = target.x - cur.x;
        const dy = target.y - cur.y;
        // Orienta o personagem (e a arma) pela direcao do movimento.
        const mag = Math.hypot(dx, dy);
        if (mag > 0.8) this.headings.set(p.playerId, { x: dx / mag, y: dy / mag });
        cur.x += dx * 0.25;
        cur.y += dy * 0.25;
      }
    }
    const ids = new Set(s.players.map((p) => p.playerId));
    for (const id of [...this.renderPos.keys()]) {
      if (!ids.has(id)) {
        this.renderPos.delete(id);
        this.headings.delete(id);
      }
    }

    // Reconstrucao dos projeteis a partir dos eventos de dano de cada snapshot.
    if (s.tick !== this.lastFxTick) {
      this.lastFxTick = s.tick;
      this.bullets.onSnapshot(s, this.headings, session.myId);
    }

    this.dyn.clear();
    drawWorld(this.dyn, s, this.renderPos, this.headings, session.myId);

    const me = this.renderPos.get(session.myId);
    if (me) this.follow.setPosition(me.x, me.y);

    this.updateHud(s);
    this.updateEndScreen(s);
  }

  /** Dispara o efeito visual do tiro do jogador local (responsivo). */
  private tryLocalFire(input: PlayerInput) {
    const me = this.state?.players.find((p) => p.playerId === session.myId && p.isAlive);
    if (!me) return;
    let to: { x: number; y: number } | undefined;
    if (input.targetPlayerId) {
      const tgt = this.state!.players.find((p) => p.playerId === input.targetPlayerId);
      if (tgt) to = { x: tgt.x, y: tgt.y };
    }
    if (!to) {
      const range = 14;
      to = { x: me.x + input.aimX * range, y: me.y + input.aimY * range };
    }
    this.bullets.localFire({ x: me.x, y: me.y }, to, me.weapon, this.headings, session.myId);
  }

  private setStatus(status: RealtimeStatus) {
    const el = document.getElementById('mode');
    if (!el) return;
    if (status === 'live') {
      el.textContent = 'AO VIVO';
      el.className = 'badge badge--live';
    } else if (status === 'connecting') {
      el.textContent = 'conectando…';
      el.className = 'badge badge--off';
    } else {
      el.textContent = 'OFFLINE (mock)';
      el.className = 'badge badge--off';
    }
  }

  private updateHud(s: GameState) {
    const me = s.players.find((p) => p.playerId === session.myId);
    const alive = s.players.filter((p) => p.isAlive).length;
    const set = (id: string, text: string) => {
      const el = document.getElementById(id);
      if (el) el.textContent = text;
    };
    set('alive', `${alive}/${s.players.length}`);
    set('phase', `${Number(s.safeZone.phase) + 1}/${PHASES}`);
    set('hp', me ? `${me.health}` : '—');
    set('weapon', me ? me.weapon : '—');
    set('tick', `${s.tick}/${MAX_MATCH_TICKS}`);
  }

  private updateEndScreen(s: GameState) {
    if (!s.matchEnded || this.endShown) return;
    this.endShown = true;

    const end = document.getElementById('end-screen') as HTMLElement | null;
    const title = document.getElementById('end-title');
    const ranking = document.getElementById('end-ranking');
    const summary = document.getElementById('end-summary');
    if (!end || !title || !ranking || !summary) return;

    const entries = s.ranking.length > 0 ? s.ranking : [...s.players]
      .sort((a, b) => Number(b.isAlive) - Number(a.isAlive) || b.health - a.health)
      .map((p, i) => ({
        playerId: p.playerId,
        place: i + 1,
        isAlive: p.isAlive,
        health: p.health,
        eliminations: p.eliminations,
        damageDealt: p.damageDealt,
        survivedTicks: p.survivedTicks,
      }));
    const mine = entries.find((e) => e.playerId === session.myId);

    title.textContent = mine?.place === 1 ? 'Vitoria!' : 'Ranking final';
    ranking.replaceChildren();
    for (const entry of entries.slice(0, 10)) {
      const li = document.createElement('li');
      li.className = entry.playerId === session.myId ? 'end-ranking__row end-ranking__row--me' : 'end-ranking__row';
      const seconds = Math.floor(Number(entry.survivedTicks) / SERVER_TICK_HZ);
      li.textContent = `#${entry.place} ${entry.playerId}${entry.playerId === session.myId ? ' (voce)' : ''} · ${entry.eliminations} elim · ${entry.damageDealt} dano · ${seconds}s`;
      ranking.appendChild(li);
    }
    summary.textContent = mine
      ? `Sua colocacao: #${mine.place} com ${mine.eliminations} eliminacao(oes).`
      : 'Partida concluida pelo servidor.';
    end.hidden = false;
  }

  private currentPlayerWorld(): { x: number; y: number } | undefined {
    const me = this.state?.players.find((p) => p.playerId === session.myId);
    return me ? { x: me.x, y: me.y } : undefined;
  }

  private withAutoTarget(input: PlayerInput): PlayerInput {
    if (!input.isAttacking || !this.state) return input;
    const me = this.state.players.find((p) => p.playerId === session.myId && p.isAlive);
    if (!me) return input;

    let nearest: { id: string; d: number; dx: number; dy: number } | null = null;
    for (const p of this.state.players) {
      if (!p.isAlive || p.playerId === session.myId) continue;
      const dx = p.x - me.x;
      const dy = p.y - me.y;
      const d = Math.hypot(dx, dy);
      if (!nearest || d < nearest.d) nearest = { id: p.playerId, d, dx, dy };
    }
    if (!nearest || nearest.d <= 0) return input;
    return {
      ...input,
      targetPlayerId: nearest.id,
      aimX: nearest.dx / nearest.d,
      aimY: nearest.dy / nearest.d,
    };
  }
}
