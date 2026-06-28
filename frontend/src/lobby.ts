// Cliente HTTP da API de salas exposta pelo Gateway (/v1/rooms/*), traduzida
// para o LobbyService gRPC via grpc-gateway. JSON em camelCase.

export type RoomStatus =
  | 'ROOM_STATUS_UNSPECIFIED'
  | 'ROOM_STATUS_WAITING'
  | 'ROOM_STATUS_STARTED'
  | 'ROOM_STATUS_CLOSED';

export interface RoomPlayer {
  playerId: string;
  playerName: string;
  ready: boolean;
}

export interface RoomResponse {
  roomId: string;
  status: RoomStatus;
  ownerId: string;
  players: RoomPlayer[];
  maxPlayers: number;
  joinUrl: string;
}

async function call<T>(path: string, body?: unknown): Promise<T> {
  const res = await fetch(path, {
    method: body === undefined ? 'GET' : 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  if (!res.ok) {
    let detail = '';
    try {
      const data = await res.json();
      detail = (data && (data.message || data.error)) || '';
    } catch {
      /* sem corpo JSON */
    }
    throw new Error(detail || `HTTP ${res.status}`);
  }
  return (await res.json()) as T;
}

export function createRoom(ownerName: string, maxPlayers?: number): Promise<RoomResponse> {
  return call<RoomResponse>('/v1/rooms', { ownerName, maxPlayers: maxPlayers ?? 0 });
}

export function joinRoom(roomId: string, playerName: string): Promise<RoomResponse> {
  return call<RoomResponse>(`/v1/rooms/${encodeURIComponent(roomId)}/join`, { playerName });
}

export function getRoom(roomId: string): Promise<RoomResponse> {
  return call<RoomResponse>(`/v1/rooms/${encodeURIComponent(roomId)}`);
}

export function setReady(roomId: string, playerId: string, ready: boolean): Promise<RoomResponse> {
  return call<RoomResponse>(`/v1/rooms/${encodeURIComponent(roomId)}/ready`, { playerId, ready });
}

export function startRoom(roomId: string, playerId: string): Promise<RoomResponse> {
  return call<RoomResponse>(`/v1/rooms/${encodeURIComponent(roomId)}/start`, { playerId });
}
