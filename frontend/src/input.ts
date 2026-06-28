// Controlador de input: teclado (WASD/setas, espaço, E) + joystick e botões DOM
// (touch/mouse). Produz um PlayerInput por amostragem.
import type { PlayerInput } from './types';
import { session } from './session';

export class Input {
  private keys = new Set<string>();
  private joy = { active: false, x: 0, y: 0 };
  private attackBtn = false;
  private openBtn = false;
  private lastAim = { x: 1, y: 0 };
  private seq = 0;

  constructor(base: HTMLElement, thumb: HTMLElement, attack: HTMLElement, open: HTMLElement) {
    addEventListener('keydown', (e) => this.keys.add(e.key.toLowerCase()));
    addEventListener('keyup', (e) => this.keys.delete(e.key.toLowerCase()));
    this.bindJoystick(base, thumb);
    this.bindHold(attack, (v) => (this.attackBtn = v));
    this.bindHold(open, (v) => (this.openBtn = v));
  }

  private bindHold(el: HTMLElement, set: (v: boolean) => void) {
    const on = (e: Event) => {
      e.preventDefault();
      set(true);
    };
    const off = () => set(false);
    el.addEventListener('pointerdown', on);
    el.addEventListener('pointerup', off);
    el.addEventListener('pointerleave', off);
    el.addEventListener('pointercancel', off);
  }

  private bindJoystick(base: HTMLElement, thumb: HTMLElement) {
    const R = 46;
    let id = -1;
    let cx = 0;
    let cy = 0;
    const start = (e: PointerEvent) => {
      e.preventDefault();
      id = e.pointerId;
      const r = base.getBoundingClientRect();
      cx = r.left + r.width / 2;
      cy = r.top + r.height / 2;
      this.joy.active = true;
      move(e);
    };
    const move = (e: PointerEvent) => {
      if (!this.joy.active || e.pointerId !== id) return;
      let dx = e.clientX - cx;
      let dy = e.clientY - cy;
      const d = Math.hypot(dx, dy);
      const k = d > R ? R / d : 1;
      dx *= k;
      dy *= k;
      thumb.style.transform = `translate(${dx}px, ${dy}px)`;
      this.joy.x = dx / R;
      this.joy.y = -dy / R; // tela y para baixo -> mundo y para cima
    };
    const end = (e: PointerEvent) => {
      if (e.pointerId !== id) return;
      this.joy.active = false;
      this.joy.x = 0;
      this.joy.y = 0;
      thumb.style.transform = 'translate(0, 0)';
      id = -1;
    };
    base.addEventListener('pointerdown', start);
    addEventListener('pointermove', move);
    addEventListener('pointerup', end);
    addEventListener('pointercancel', end);
  }

  sample(): PlayerInput {
    let mx = 0;
    let my = 0;
    if (this.joy.active) {
      mx = this.joy.x;
      my = this.joy.y;
    } else {
      if (this.keys.has('d') || this.keys.has('arrowright')) mx += 1;
      if (this.keys.has('a') || this.keys.has('arrowleft')) mx -= 1;
      if (this.keys.has('w') || this.keys.has('arrowup')) my += 1;
      if (this.keys.has('s') || this.keys.has('arrowdown')) my -= 1;
    }
    const isAttacking = this.attackBtn || this.keys.has(' ');
    const openChest = this.openBtn || this.keys.has('e');
    const mag = Math.hypot(mx, my);
    if (mag > 0.01) {
      this.lastAim = { x: mx / mag, y: my / mag };
    }
    return {
      playerId: session.myId,
      moveX: mx,
      moveY: my,
      isAttacking,
      openChest,
      inputSequence: ++this.seq,
      aimX: this.lastAim.x,
      aimY: this.lastAim.y,
      roomId: session.roomId,
    };
  }
}
