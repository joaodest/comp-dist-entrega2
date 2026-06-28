// Tela de lobby (Fase 3): criar sala com QR Code/URL, entrar pelo celular com o
// nome, ver jogadores e estado pronto/aguardando, e iniciar a partida.
// Resolve a Promise quando a sala entra em ROOM_STATUS_STARTED, gravando a
// sessao (roomId/myId/isOwner) usada pela partida.
import QRCode from 'qrcode';
import { session } from './session';
import {
  createRoom,
  getRoom,
  joinRoom,
  setReady,
  startRoom,
  type RoomResponse,
} from './lobby';

const POLL_MS = 1200;

function joinLink(roomId: string): string {
  return `${location.origin}/?room=${encodeURIComponent(roomId)}`;
}

function statusLabel(status: RoomResponse['status']): string {
  switch (status) {
    case 'ROOM_STATUS_WAITING':
      return 'aguardando';
    case 'ROOM_STATUS_STARTED':
      return 'em partida';
    case 'ROOM_STATUS_CLOSED':
      return 'fechada';
    default:
      return '—';
  }
}

export function runLobby(root: HTMLElement): Promise<void> {
  return new Promise<void>((resolve) => {
    const params = new URLSearchParams(location.search);
    const roomFromUrl = (params.get('room') ?? '').trim();

    let poll: number | undefined;
    const stopPoll = () => {
      if (poll !== undefined) {
        clearInterval(poll);
        poll = undefined;
      }
    };

    const enterMatch = (room: RoomResponse) => {
      stopPoll();
      session.roomId = room.roomId;
      resolve();
    };

    const showError = (msg: string) => {
      const el = document.getElementById('lobby-error');
      if (el) el.textContent = msg;
    };

    // Renderiza a sala (lista de jogadores, ready, start, QR) e faz polling.
    const showRoom = async (initial: RoomResponse) => {
      stopPoll();

      const render = (room: RoomResponse) => {
        if (room.status === 'ROOM_STATUS_STARTED') {
          enterMatch(room);
          return;
        }
        if (room.status === 'ROOM_STATUS_CLOSED') {
          stopPoll();
          showLanding('A sala foi fechada.');
          return;
        }

        const me = room.players.find((p) => p.playerId === session.myId);
        const iAmReady = me?.ready ?? false;
        const isOwner = room.ownerId === session.myId;
        session.isOwner = isOwner;

        const list = room.players
          .map((p) => {
            const tags: string[] = [];
            if (p.playerId === room.ownerId) tags.push('<span class="lb-tag lb-tag--owner">dono</span>');
            tags.push(
              p.ready
                ? '<span class="lb-tag lb-tag--ready">pronto</span>'
                : '<span class="lb-tag">aguardando</span>',
            );
            const mineMark = p.playerId === session.myId ? ' (você)' : '';
            return `<li class="lb-player"><span>${escapeHtml(p.playerName)}${mineMark}</span><span>${tags.join('')}</span></li>`;
          })
          .join('');

        root.innerHTML = `
          <div class="lobby">
            <div class="lobby__card">
              <h1 class="lobby__title">Sala ${escapeHtml(room.roomId)}</h1>
              <p class="lobby__sub">${room.players.length}/${room.maxPlayers} jogadores · ${statusLabel(room.status)}</p>
              <div class="lobby__grid">
                <div>
                  <ul class="lb-players">${list}</ul>
                  <div class="lobby__actions">
                    <button id="lb-ready" class="lb-btn ${iAmReady ? '' : 'lb-btn--primary'}">${iAmReady ? 'Cancelar pronto' : 'Estou pronto'}</button>
                    ${isOwner ? '<button id="lb-start" class="lb-btn lb-btn--go">Iniciar partida</button>' : ''}
                  </div>
                  <p id="lobby-error" class="lobby__error"></p>
                </div>
                <div class="lobby__qr">
                  <img id="lb-qr" alt="QR Code da sala" width="180" height="180" />
                  <p class="lobby__hint">Escaneie ou compartilhe o link para entrar pelo celular:</p>
                  <code class="lobby__link">${escapeHtml(joinLink(room.roomId))}</code>
                  <button id="lb-copy" class="lb-btn lb-btn--ghost">Copiar link</button>
                </div>
              </div>
            </div>
          </div>`;

        void QRCode.toDataURL(joinLink(room.roomId), { width: 180, margin: 1 })
          .then((url) => {
            const img = document.getElementById('lb-qr') as HTMLImageElement | null;
            if (img) img.src = url;
          })
          .catch(() => {
            /* QR opcional: o link textual ja basta */
          });

        document.getElementById('lb-ready')?.addEventListener('click', async () => {
          try {
            const updated = await setReady(room.roomId, session.myId, !iAmReady);
            render(updated);
          } catch (e) {
            showError(`Não foi possível atualizar: ${(e as Error).message}`);
          }
        });

        document.getElementById('lb-start')?.addEventListener('click', async () => {
          try {
            const updated = await startRoom(room.roomId, session.myId);
            render(updated);
          } catch (e) {
            showError(`Não foi possível iniciar: ${(e as Error).message}`);
          }
        });

        document.getElementById('lb-copy')?.addEventListener('click', () => {
          void navigator.clipboard?.writeText(joinLink(room.roomId)).catch(() => {});
        });
      };

      render(initial);

      poll = window.setInterval(async () => {
        try {
          const room = await getRoom(initial.roomId);
          render(room);
        } catch {
          /* mantem a ultima renderizacao se o Gateway oscilar */
        }
      }, POLL_MS);
    };

    // Tela inicial: criar sala, entrada rapida (solo) ou entrar por ID.
    const showLanding = (notice = '') => {
      stopPoll();
      root.innerHTML = `
        <div class="lobby">
          <div class="lobby__card">
            <h1 class="lobby__title">Voxel Royale</h1>
            <p class="lobby__sub">Crie uma sala e chame os jogadores por QR Code.</p>
            ${notice ? `<p class="lobby__error">${escapeHtml(notice)}</p>` : ''}
            <div class="lobby__form">
              <label class="lobby__label">Seu nome</label>
              <input id="lb-name" class="lobby__input" maxlength="20" placeholder="ex.: Ana" />
              <div class="lobby__row">
                <input id="lb-max" class="lobby__input" type="number" min="2" max="50" value="10" />
                <span class="lobby__hint">máx. de jogadores</span>
              </div>
              <button id="lb-create" class="lb-btn lb-btn--primary">Criar sala</button>
              <button id="lb-quick" class="lb-btn lb-btn--ghost">Jogar agora (sala solo)</button>
            </div>
            <div class="lobby__sep">ou entrar em uma sala</div>
            <div class="lobby__row">
              <input id="lb-room" class="lobby__input" placeholder="ID da sala (ex.: room-1)" />
              <button id="lb-join" class="lb-btn">Entrar</button>
            </div>
            <p id="lobby-error" class="lobby__error"></p>
          </div>
        </div>`;

      const nameInput = document.getElementById('lb-name') as HTMLInputElement;

      document.getElementById('lb-create')?.addEventListener('click', async () => {
        const name = nameInput.value.trim();
        if (!name) return showError('Informe seu nome.');
        const max = Number((document.getElementById('lb-max') as HTMLInputElement).value) || 0;
        try {
          const room = await createRoom(name, max);
          session.myId = room.ownerId;
          session.isOwner = true;
          await showRoom(room);
        } catch (e) {
          showError(`Falha ao criar sala: ${(e as Error).message}`);
        }
      });

      document.getElementById('lb-quick')?.addEventListener('click', async () => {
        const name = nameInput.value.trim() || 'Você';
        try {
          const room = await createRoom(name, 0);
          session.myId = room.ownerId;
          session.isOwner = true;
          const started = await startRoom(room.roomId, room.ownerId);
          enterMatch(started);
        } catch (e) {
          showError(`Falha no modo rápido: ${(e as Error).message}`);
        }
      });

      document.getElementById('lb-join')?.addEventListener('click', async () => {
        const roomId = (document.getElementById('lb-room') as HTMLInputElement).value.trim();
        const name = nameInput.value.trim();
        if (!roomId) return showError('Informe o ID da sala.');
        if (!name) return showError('Informe seu nome.');
        await doJoin(roomId, name);
      });
    };

    // Tela de entrada quando se chega por QR Code (?room=...): pede so o nome.
    const showJoinByUrl = (roomId: string) => {
      root.innerHTML = `
        <div class="lobby">
          <div class="lobby__card">
            <h1 class="lobby__title">Entrar na sala</h1>
            <p class="lobby__sub">Sala <b>${escapeHtml(roomId)}</b></p>
            <div class="lobby__form">
              <label class="lobby__label">Seu nome</label>
              <input id="lb-name" class="lobby__input" maxlength="20" placeholder="ex.: Bruno" />
              <button id="lb-join" class="lb-btn lb-btn--primary">Entrar</button>
              <button id="lb-back" class="lb-btn lb-btn--ghost">Criar uma sala</button>
            </div>
            <p id="lobby-error" class="lobby__error"></p>
          </div>
        </div>`;

      document.getElementById('lb-join')?.addEventListener('click', async () => {
        const name = (document.getElementById('lb-name') as HTMLInputElement).value.trim();
        if (!name) return showError('Informe seu nome.');
        await doJoin(roomId, name);
      });
      document.getElementById('lb-back')?.addEventListener('click', () => showLanding());
    };

    const doJoin = async (roomId: string, name: string) => {
      try {
        const room = await joinRoom(roomId, name);
        // O servidor anexa o novo jogador ao fim da lista.
        const me = room.players[room.players.length - 1];
        session.myId = me?.playerId ?? session.myId;
        session.isOwner = room.ownerId === session.myId;
        await showRoom(room);
      } catch (e) {
        showError(`Falha ao entrar: ${(e as Error).message}`);
      }
    };

    if (roomFromUrl) {
      showJoinByUrl(roomFromUrl);
    } else {
      showLanding();
    }
  });
}

function escapeHtml(value: string): string {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}
