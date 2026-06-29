// Efeitos visuais de combate (puramente do cliente). O servidor resolve os
// ataques de forma instantanea (hitscan) e so reporta dano/eliminacoes no
// snapshot; aqui reconstruimos os projeteis a partir das variacoes de dano
// entre snapshots e dos disparos do proprio jogador, animando tracos de bala,
// faiscas de impacto, clarao do cano e explosao de eliminacao.
import Phaser from 'phaser';
import type { GameState } from './types';
import { PX, worldToPx } from './config';

type Heading = { x: number; y: number };

interface WeaponFx {
  color: number;
  core: number;
  width: number;
  pellets: number;
  speed: number; // px por ms
  spread: number; // radianos de dispersao (shotgun)
  cooldownMs: number;
}

const WEAPON_FX: Record<string, WeaponFx> = {
  pistol: { color: 0xffe08a, core: 0xffffff, width: 3, pellets: 1, speed: 2.8, spread: 0, cooldownMs: 110 },
  rifle: { color: 0x9fe8ff, core: 0xffffff, width: 2.6, pellets: 1, speed: 3.8, spread: 0, cooldownMs: 90 },
  shotgun: { color: 0xffb15e, core: 0xfff0d6, width: 3, pellets: 5, speed: 2.3, spread: 0.5, cooldownMs: 220 },
};

function fxFor(weapon: string): WeaponFx {
  return WEAPON_FX[weapon] ?? WEAPON_FX.pistol;
}

interface Tracer {
  x0: number;
  y0: number;
  x1: number;
  y1: number;
  elapsed: number;
  duration: number;
  color: number;
  core: number;
  width: number;
}

interface Spark {
  x: number;
  y: number;
  elapsed: number;
  ttl: number;
  color: number;
  radius: number;
  rays: number;
  lethal: boolean;
}

interface Flash {
  x: number;
  y: number;
  dirX: number;
  dirY: number;
  elapsed: number;
  ttl: number;
  color: number;
  size: number;
}

interface Snap {
  dealt: number;
  taken: number;
  health: number;
  alive: boolean;
  x: number;
  y: number;
  weapon: string;
}

/** Gera e anima os efeitos de tiro do cliente. */
export class Bullets {
  private tracers: Tracer[] = [];
  private sparks: Spark[] = [];
  private flashes: Flash[] = [];
  private prev = new Map<string, Snap>();
  private primed = false;
  private localCooldownUntil = 0;

  /** Disparo imediato do jogador local (responsivo, funciona ate offline). */
  localFire(
    fromWorld: { x: number; y: number },
    toWorld: { x: number; y: number },
    weapon: string,
    headings: Map<string, Heading>,
    myId: string,
    hitRock = false,
  ): void {
    const now = performance.now();
    const fx = fxFor(weapon);
    if (now < this.localCooldownUntil) return;
    this.localCooldownUntil = now + fx.cooldownMs;

    const from = worldToPx(fromWorld.x, fromWorld.y);
    const to = worldToPx(toWorld.x, toWorld.y);
    let dx = to.x - from.x;
    let dy = to.y - from.y;
    const d = Math.hypot(dx, dy) || 1;
    dx /= d;
    dy /= d;
    headings.set(myId, { x: dx, y: dy });
    this.fire(from.x, from.y, to.x, to.y, fx);
    if (hitRock) this.spawnImpact(to.x, to.y, fx.color, false);
  }

  /**
   * Detecta ataques a partir das variacoes de dano entre snapshots e cria os
   * projeteis correspondentes. Ignora o dano causado pelo jogador local (ja
   * desenhado em localFire) para nao duplicar o traco.
   */
  onSnapshot(s: GameState, headings: Map<string, Heading>, myId: string): void {
    const cur = new Map<string, Snap>();
    for (const p of s.players) {
      cur.set(p.playerId, {
        dealt: p.damageDealt,
        taken: p.damageTaken,
        health: p.health,
        alive: p.isAlive,
        x: p.x,
        y: p.y,
        weapon: p.weapon,
      });
    }

    if (!this.primed) {
      this.prev = cur;
      this.primed = true;
      return;
    }

    // Vitimas: quanto cada jogador levou de dano desde o ultimo snapshot.
    const pendingTaken = new Map<string, number>();
    for (const [id, c] of cur) {
      const delta = c.taken - (this.prev.get(id)?.taken ?? c.taken);
      if (delta > 0) pendingTaken.set(id, delta);
    }

    for (const [id, c] of cur) {
      const before = this.prev.get(id);
      if (!before) continue;
      const dealtDelta = c.dealt - before.dealt;
      if (dealtDelta <= 0) continue;

      // Casa o atacante com a vitima mais proxima que ainda tem dano pendente.
      const victimId = this.matchVictim(id, c, pendingTaken, cur);
      if (!victimId) continue;
      const victim = cur.get(victimId)!;
      pendingTaken.set(victimId, (pendingTaken.get(victimId) ?? 0) - dealtDelta);

      const from = worldToPx(c.x, c.y);
      const to = worldToPx(victim.x, victim.y);
      let dx = to.x - from.x;
      let dy = to.y - from.y;
      const dist = Math.hypot(dx, dy) || 1;
      dx /= dist;
      dy /= dist;
      headings.set(id, { x: dx, y: dy });

      const victimWasAlive = this.prev.get(victimId)?.alive ?? true;
      const lethal = victimWasAlive && !victim.alive;
      // O jogador local ja desenhou seu proprio traco em localFire.
      if (id !== myId) this.fire(from.x, from.y, to.x, to.y, fxFor(c.weapon));

      this.spawnImpact(to.x, to.y, fxFor(c.weapon).color, lethal);
    }

    this.prev = cur;
  }

  private matchVictim(
    attackerId: string,
    attacker: Snap,
    pendingTaken: Map<string, number>,
    cur: Map<string, Snap>,
  ): string | null {
    let best: string | null = null;
    let bestDist = Infinity;
    for (const [id, remaining] of pendingTaken) {
      if (id === attackerId || remaining <= 0) continue;
      const v = cur.get(id);
      if (!v) continue;
      const dist = Math.hypot(v.x - attacker.x, v.y - attacker.y);
      if (dist < bestDist) {
        bestDist = dist;
        best = id;
      }
    }
    return best;
  }

  private fire(x0: number, y0: number, x1: number, y1: number, fx: WeaponFx): void {
    let dx = x1 - x0;
    let dy = y1 - y0;
    const dist = Math.hypot(dx, dy) || 1;
    dx /= dist;
    dy /= dist;

    // Recua a origem para a "ponta do cano" (corpo tem ~1 unidade de raio).
    const muzzle = PX * 0.95;
    const mx = x0 + dx * muzzle;
    const my = y0 + dy * muzzle;
    this.flashes.push({ x: mx, y: my, dirX: dx, dirY: dy, elapsed: 0, ttl: 70, color: fx.core, size: PX * 0.5 });

    for (let i = 0; i < fx.pellets; i++) {
      let adx = dx;
      let ady = dy;
      if (fx.spread > 0) {
        const a = (Math.random() - 0.5) * fx.spread;
        const cos = Math.cos(a);
        const sin = Math.sin(a);
        adx = dx * cos - dy * sin;
        ady = dx * sin + dy * cos;
      }
      const ex = fx.spread > 0 ? mx + adx * dist : x1;
      const ey = fx.spread > 0 ? my + ady * dist : y1;
      const duration = Math.max(45, Math.hypot(ex - mx, ey - my) / fx.speed);
      this.tracers.push({ x0: mx, y0: my, x1: ex, y1: ey, elapsed: 0, duration, color: fx.color, core: fx.core, width: fx.width });
    }
  }

  private spawnImpact(x: number, y: number, color: number, lethal: boolean): void {
    this.sparks.push({
      x,
      y,
      elapsed: 0,
      ttl: lethal ? 420 : 200,
      color: lethal ? 0xff5d6c : color,
      radius: lethal ? PX * 2.4 : PX * 0.9,
      rays: lethal ? 10 : 6,
      lethal,
    });
  }

  update(deltaMs: number): void {
    for (const t of this.tracers) t.elapsed += deltaMs;
    for (const s of this.sparks) s.elapsed += deltaMs;
    for (const f of this.flashes) f.elapsed += deltaMs;
    this.tracers = this.tracers.filter((t) => t.elapsed < t.duration);
    this.sparks = this.sparks.filter((s) => s.elapsed < s.ttl);
    this.flashes = this.flashes.filter((f) => f.elapsed < f.ttl);
  }

  draw(g: Phaser.GameObjects.Graphics): void {
    g.clear();

    // Clarao do cano.
    for (const f of this.flashes) {
      const k = 1 - f.elapsed / f.ttl;
      const r = f.size * (0.6 + k * 0.6);
      g.fillStyle(f.color, 0.85 * k);
      g.fillCircle(f.x, f.y, r);
      g.fillStyle(0xffd27a, 0.5 * k);
      g.fillCircle(f.x, f.y, r * 0.55);
    }

    // Tracos de bala (cabeca brilhante + rastro).
    for (const t of this.tracers) {
      const p = t.elapsed / t.duration;
      const hx = t.x0 + (t.x1 - t.x0) * p;
      const hy = t.y0 + (t.y1 - t.y0) * p;
      const trail = 26;
      let dx = t.x1 - t.x0;
      let dy = t.y1 - t.y0;
      const d = Math.hypot(dx, dy) || 1;
      dx /= d;
      dy /= d;
      const tx = hx - dx * trail;
      const ty = hy - dy * trail;
      g.lineStyle(t.width + 2, t.color, 0.35);
      g.lineBetween(tx, ty, hx, hy);
      g.lineStyle(t.width, t.core, 0.95);
      g.lineBetween(hx - dx * trail * 0.45, hy - dy * trail * 0.45, hx, hy);
      g.fillStyle(t.core, 1);
      g.fillCircle(hx, hy, t.width * 0.9);
    }

    // Faiscas de impacto / explosao de eliminacao.
    for (const s of this.sparks) {
      const k = s.elapsed / s.ttl;
      const inv = 1 - k;
      const r = s.radius * (0.3 + k * 1.1);
      g.lineStyle(s.lethal ? 4 : 2.5, s.color, inv);
      g.strokeCircle(s.x, s.y, r);
      for (let i = 0; i < s.rays; i++) {
        const a = (i / s.rays) * Math.PI * 2 + k;
        const r0 = r * 0.5;
        const r1 = r * (s.lethal ? 1.3 : 1.0);
        g.lineBetween(s.x + Math.cos(a) * r0, s.y + Math.sin(a) * r0, s.x + Math.cos(a) * r1, s.y + Math.sin(a) * r1);
      }
      if (s.lethal) {
        g.fillStyle(0xffd27a, inv * 0.5);
        g.fillCircle(s.x, s.y, r * 0.5);
      }
    }
  }

  reset(): void {
    this.tracers = [];
    this.sparks = [];
    this.flashes = [];
    this.prev.clear();
    this.primed = false;
  }
}
