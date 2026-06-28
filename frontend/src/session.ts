// Sessao do jogador definida pela tela de lobby antes de entrar na partida.
// roomId roteia o input para a partida da sala (StreamMatch com room_id);
// myId e o playerId atribuido pelo Lobby (owner ou jogador que entrou).
import { MY_ID } from './config';

export interface Session {
  roomId: string;
  myId: string;
  isOwner: boolean;
}

export const session: Session = {
  roomId: '',
  myId: MY_ID,
  isOwner: false,
};
