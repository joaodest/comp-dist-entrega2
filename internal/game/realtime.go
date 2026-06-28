package game

import (
	"context"
	"strings"
	"time"

	matchv1 "voxel-royale/gen/match"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// tickHz e a frequencia do relogio do servidor para partidas em tempo real.
	// O relogio avanca a simulacao independentemente do ritmo dos clientes:
	// inputs sao bufferizados e consumidos a cada tick (modelo authoritative).
	tickHz = 15
	// subscriberBuffer limita os snapshots em voo por assinante; quando cheio,
	// o broadcast descarta o mais novo para nao travar o relogio por um cliente
	// lento (preferimos atraso a backpressure global).
	subscriberBuffer = 8
)

// subscriber representa um cliente assinando os snapshots de uma partida via
// WatchMatch. O Gateway abre um WatchMatch por WebSocket conectado.
type subscriber struct {
	ch chan *matchv1.GameState
}

// PushInput encaminha um input do cliente para o buffer da partida (NETW-03).
// O input nao e aplicado imediatamente: o relogio do servidor o consome no
// proximo tick. O ultimo input por jogador vence (sobrescreve).
func (s *Server) PushInput(_ context.Context, input *matchv1.PlayerInput) (*matchv1.InputAck, error) {
	if input == nil || strings.TrimSpace(input.PlayerId) == "" {
		return nil, status.Error(codes.InvalidArgument, "player_id is required")
	}
	if !isFinite(input.MoveX) || !isFinite(input.MoveY) || !isFinite(input.AimX) || !isFinite(input.AimY) {
		return nil, status.Error(codes.InvalidArgument, "movement and aim values must be finite")
	}

	playerID := strings.TrimSpace(input.PlayerId)
	matchKey := matchKeyFor(input.RoomId)

	s.mu.Lock()
	defer s.mu.Unlock()

	match := s.matches[matchKey]
	if match == nil {
		match = newMatchState()
		s.matches[matchKey] = match
	}
	match.ensurePlayer(playerID)
	match.pendingInputs[playerID] = input

	return &matchv1.InputAck{Accepted: true, AppliedSequence: input.InputSequence}, nil
}

// WatchMatch assina os snapshots publicados pelo relogio do servidor e os envia
// pelo stream gRPC (NETW-04). O Gateway repassa cada snapshot ao WebSocket do
// jogador. O relogio da partida e iniciado sob demanda no primeiro assinante e
// parado quando o ultimo assinante desconecta.
func (s *Server) WatchMatch(req *matchv1.WatchMatchRequest, stream matchv1.GameService_WatchMatchServer) error {
	if req == nil {
		return status.Error(codes.InvalidArgument, "request is required")
	}
	matchKey := matchKeyFor(req.RoomId)
	playerID := strings.TrimSpace(req.PlayerId)

	sub := &subscriber{ch: make(chan *matchv1.GameState, subscriberBuffer)}

	s.mu.Lock()
	match := s.matches[matchKey]
	if match == nil {
		match = newMatchState()
		s.matches[matchKey] = match
	}
	if playerID != "" {
		match.ensurePlayer(playerID)
	}
	match.subs[sub] = struct{}{}
	initial := match.snapshot()
	s.ensureClockLocked(matchKey)
	s.mu.Unlock()

	defer s.removeSubscriber(matchKey, sub)

	if err := stream.Send(initial); err != nil {
		return err
	}

	ctx := stream.Context()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case snapshot := <-sub.ch:
			if err := stream.Send(snapshot); err != nil {
				return err
			}
		}
	}
}

// ensureClockLocked inicia a goroutine do relogio da partida caso ainda nao
// esteja rodando. Deve ser chamado com s.mu travado.
func (s *Server) ensureClockLocked(matchKey string) {
	match := s.matches[matchKey]
	if match == nil || match.clockRunning {
		return
	}
	match.clockRunning = true
	go s.runClock(matchKey)
}

// runClock avanca a simulacao da sala no proprio relogio do servidor e publica
// um snapshot por tick a todos os assinantes. Encerra-se quando a sala some ou
// fica sem assinantes (deixando a partida em memoria, pronta para reassinar).
func (s *Server) runClock(matchKey string) {
	ticker := time.NewTicker(time.Second / tickHz)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		match := s.matches[matchKey]
		if match == nil {
			s.mu.Unlock()
			return
		}
		if len(match.subs) == 0 {
			match.clockRunning = false
			s.mu.Unlock()
			return
		}
		// Apos o fim da partida o relogio para de avancar ticks, mas segue
		// publicando o snapshot final para que assinantes vejam o ranking.
		if !match.matchEnded {
			match.advanceTick()
		}
		snapshot := match.snapshot()
		targets := make([]*subscriber, 0, len(match.subs))
		for sub := range match.subs {
			targets = append(targets, sub)
		}
		s.mu.Unlock()

		for _, sub := range targets {
			select {
			case sub.ch <- snapshot:
			default:
			}
		}
	}
}

// removeSubscriber desliga um assinante (WebSocket encerrado ou erro de envio).
func (s *Server) removeSubscriber(matchKey string, sub *subscriber) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if match := s.matches[matchKey]; match != nil {
		delete(match.subs, sub)
	}
}

// advanceTick e o passo autoritativo do relogio: aplica o input bufferizado de
// cada jogador, resolve zona segura, sobrevivencia e fim de partida. Difere do
// StreamMatch (unario, legado) por consumir inputs bufferizados em vez de um
// input por requisicao.
func (m *matchState) advanceTick() {
	m.tick++
	for _, playerID := range m.playerIDs {
		player := m.players[playerID]
		if player == nil || !player.alive {
			continue
		}
		input := m.pendingInputs[playerID]
		if input == nil {
			continue
		}
		if isStaleInput(player, input) {
			continue
		}
		applyInputSequence(player, input)
		m.movePlayer(player, input.MoveX, input.MoveY)
		if input.OpenChest {
			m.openNearestChest(player)
		}
		if input.IsAttacking {
			m.attack(player, input)
		}
	}
	m.applySafeZoneDamage()
	m.refreshSurvivalTicks()
	m.updateMatchEnd()
}

// matchKeyFor normaliza a chave da partida a partir do room_id; vazio cai na
// partida global (compatibilidade com o modo demo de um jogador).
func matchKeyFor(roomID string) string {
	if key := strings.TrimSpace(roomID); key != "" {
		return key
	}
	return globalMatchKey
}
