import Phaser from 'phaser';
import type { GameState } from './types';
import { Input } from './input';
import type { Driver, Mode } from './net';
import { LiveDriver, OfflineDriver } from './net';
import { drawTerrain, drawWorld, GRASS } from './ioRender';
import { SEND_MS, worldToPx, PHASES, MAX_MATCH_TICKS } from './config';
import { session } from './session';

export class GameScene extends Phaser.Scene {
  private controls: Input;
  private driver: Driver = new LiveDriver();
  private state: GameState | null = null;
  private renderPos = new Map<string, { x: number; y: number }>();
  private terrain!: Phaser.GameObjects.Graphics;
  private dyn!: Phaser.GameObjects.Graphics;
  private follow!: Phaser.GameObjects.Arc;
  private accum = 0;
  private pending = false;
  private triedOffline = false;

  constructor(controls: Input) {
    super('game');
    this.controls = controls;
  }

  create() {
    this.cameras.main.setBackgroundColor(GRASS);
    this.terrain = this.add.graphics();
    drawTerrain(this.terrain);
    this.dyn = this.add.graphics();

    const center = worldToPx(0, 0);
    this.follow = this.add.circle(center.x, center.y, 1, 0x000000, 0);
    this.cameras.main.setZoom(1.3);
    this.cameras.main.startFollow(this.follow, true, 0.15, 0.15);
  }

  update(_time: number, delta: number) {
    this.accum += delta;
    if (this.accum >= SEND_MS && !this.pending) {
      this.accum = 0;
      void this.sendStep();
    }
    if (this.state) this.draw();
  }

  private async sendStep() {
    this.pending = true;
    const input = this.controls.sample();
    try {
      this.state = await this.driver.step(input);
      this.setMode(this.driver.mode);
    } catch {
      if (!this.triedOffline) {
        this.triedOffline = true;
        this.driver = new OfflineDriver(this.renderPos.get(session.myId));
        try {
          this.state = await this.driver.step(input);
          this.setMode('offline');
        } catch {
          /* ignore */
        }
      }
    } finally {
      this.pending = false;
    }
  }

  private draw() {
    const s = this.state!;
    for (const p of s.players) {
      const target = worldToPx(p.x, p.y);
      const cur = this.renderPos.get(p.playerId);
      if (!cur) this.renderPos.set(p.playerId, target);
      else {
        cur.x += (target.x - cur.x) * 0.25;
        cur.y += (target.y - cur.y) * 0.25;
      }
    }
    const ids = new Set(s.players.map((p) => p.playerId));
    for (const id of [...this.renderPos.keys()]) if (!ids.has(id)) this.renderPos.delete(id);

    this.dyn.clear();
    drawWorld(this.dyn, s, this.renderPos, session.myId);

    const me = this.renderPos.get(session.myId);
    if (me) this.follow.setPosition(me.x, me.y);

    this.updateHud(s);
  }

  private setMode(mode: Mode) {
    const el = document.getElementById('mode');
    if (!el) return;
    el.textContent = mode === 'live' ? 'AO VIVO' : 'OFFLINE (mock)';
    el.className = 'badge ' + (mode === 'live' ? 'badge--live' : 'badge--off');
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
}
