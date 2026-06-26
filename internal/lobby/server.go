package lobby

import (
	"context"
	"fmt"
	"strings"
	"sync"

	lobbyv1 "voxel-royale/gen/lobby"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const defaultMaxPlayers = 50

type room struct {
	id           string
	ownerID      string
	ownerName    string
	status       lobbyv1.RoomStatus
	maxPlayers   int32
	players      []*lobbyv1.Player
	playerSet    map[string]bool
	nextPlayerID uint64
}

type Server struct {
	lobbyv1.UnimplementedLobbyServiceServer

	mu     sync.RWMutex
	rooms  map[string]*room
	nextID uint64
}

func NewServer() *Server {
	return &Server{
		rooms: make(map[string]*room),
	}
}

func (s *Server) CreateRoom(_ context.Context, req *lobbyv1.CreateRoomRequest) (*lobbyv1.RoomResponse, error) {
	if req == nil || strings.TrimSpace(req.OwnerName) == "" {
		return nil, status.Error(codes.InvalidArgument, "owner_name is required")
	}

	maxPlayers := req.MaxPlayers
	if maxPlayers <= 0 {
		maxPlayers = defaultMaxPlayers
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextID++
	roomID := fmt.Sprintf("room-%d", s.nextID)
	playerID := fmt.Sprintf("player-%d-1", s.nextID)

	owner := &lobbyv1.Player{
		PlayerId:   playerID,
		PlayerName: strings.TrimSpace(req.OwnerName),
	}

	r := &room{
		id:           roomID,
		ownerID:      playerID,
		ownerName:    strings.TrimSpace(req.OwnerName),
		status:       lobbyv1.RoomStatus_ROOM_STATUS_WAITING,
		maxPlayers:   maxPlayers,
		players:      []*lobbyv1.Player{owner},
		playerSet:    map[string]bool{playerID: true},
		nextPlayerID: 2,
	}
	s.rooms[roomID] = r

	return roomToResponse(r), nil
}

func (s *Server) JoinRoom(_ context.Context, req *lobbyv1.JoinRoomRequest) (*lobbyv1.RoomResponse, error) {
	roomID, err := requireTrimmed(req.GetRoomId(), "room_id")
	if err != nil {
		return nil, err
	}
	playerName, err := requireTrimmed(req.GetPlayerName(), "player_name")
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	r, ok := s.rooms[roomID]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "room %s not found", roomID)
	}
	if r.status != lobbyv1.RoomStatus_ROOM_STATUS_WAITING {
		return nil, status.Errorf(codes.FailedPrecondition, "room %s is not accepting players", roomID)
	}
	if int32(len(r.players)) >= r.maxPlayers {
		return nil, status.Errorf(codes.FailedPrecondition, "room %s is full", roomID)
	}

	playerID := fmt.Sprintf("player-%s-%d", roomID, r.nextPlayerID)
	r.nextPlayerID++

	player := &lobbyv1.Player{
		PlayerId:   playerID,
		PlayerName: playerName,
	}
	r.players = append(r.players, player)
	r.playerSet[playerID] = true

	return roomToResponse(r), nil
}

func (s *Server) GetRoom(_ context.Context, req *lobbyv1.GetRoomRequest) (*lobbyv1.RoomResponse, error) {
	roomID, err := requireTrimmed(req.GetRoomId(), "room_id")
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	r, ok := s.rooms[roomID]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "room %s not found", roomID)
	}

	return roomToResponse(r), nil
}

func (s *Server) StartRoom(_ context.Context, req *lobbyv1.StartRoomRequest) (*lobbyv1.RoomResponse, error) {
	roomID, err := requireTrimmed(req.GetRoomId(), "room_id")
	if err != nil {
		return nil, err
	}
	playerID, err := requireTrimmed(req.GetPlayerId(), "player_id")
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	r, ok := s.rooms[roomID]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "room %s not found", roomID)
	}
	if r.status != lobbyv1.RoomStatus_ROOM_STATUS_WAITING {
		return nil, status.Errorf(codes.FailedPrecondition, "room %s cannot be started (status: %v)", roomID, r.status)
	}
	if playerID != r.ownerID {
		return nil, status.Errorf(codes.PermissionDenied, "only the room owner can start the room")
	}

	r.status = lobbyv1.RoomStatus_ROOM_STATUS_STARTED

	return roomToResponse(r), nil
}

func (s *Server) LeaveRoom(_ context.Context, req *lobbyv1.LeaveRoomRequest) (*lobbyv1.RoomResponse, error) {
	roomID, err := requireTrimmed(req.GetRoomId(), "room_id")
	if err != nil {
		return nil, err
	}
	playerID, err := requireTrimmed(req.GetPlayerId(), "player_id")
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	r, ok := s.rooms[roomID]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "room %s not found", roomID)
	}
	if !r.playerSet[playerID] {
		return nil, status.Errorf(codes.NotFound, "player %s not found in room %s", playerID, roomID)
	}

	delete(r.playerSet, playerID)
	filtered := r.players[:0]
	for _, p := range r.players {
		if p.PlayerId != playerID {
			filtered = append(filtered, p)
		}
	}
	r.players = filtered

	if playerID == r.ownerID {
		if len(r.players) > 0 {
			r.ownerID = r.players[0].PlayerId
		} else {
			r.status = lobbyv1.RoomStatus_ROOM_STATUS_CLOSED
			r.ownerID = ""
			delete(s.rooms, roomID)
		}
	}

	resp := roomToResponse(r)
	return resp, nil
}
func (r *room) allReady() bool {
	if len(r.players) == 0 {
		return false
	}

	for _, p := range r.players {
		if !p.Ready {
			return false
		}
	}

	return true
}

func (s *Server) SetReady(_ context.Context, req *lobbyv1.SetReadyRequest) (*lobbyv1.RoomResponse, error) {
	roomID, err := requireTrimmed(req.GetRoomId(), "room_id")
	if err != nil {
		return nil, err
	}

	playerID, err := requireTrimmed(req.GetPlayerId(), "player_id")
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	r, ok := s.rooms[roomID]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "room %s not found", roomID)
	}

	if r.status != lobbyv1.RoomStatus_ROOM_STATUS_WAITING {
		return nil, status.Errorf(
			codes.FailedPrecondition,
			"room %s is not in waiting state (status: %v)",
			roomID,
			r.status,
		)
	}

	if !r.playerSet[playerID] {
		return nil, status.Errorf(
			codes.NotFound,
			"player %s not found in room %s",
			playerID,
			roomID,
		)
	}

	var player *lobbyv1.Player

	for _, p := range r.players {
		if p.PlayerId == playerID {
			player = p
			break
		}
	}

	if player == nil {
		return nil, status.Errorf(
			codes.NotFound,
			"player %s not found in room %s",
			playerID,
			roomID,
		)
	}

	player.Ready = req.Ready

	if req.Ready && r.allReady() {
		r.status = lobbyv1.RoomStatus_ROOM_STATUS_STARTED
	}

	return roomToResponse(r), nil
}

func roomToResponse(r *room) *lobbyv1.RoomResponse {
	players := make([]*lobbyv1.Player, len(r.players))
	for i, p := range r.players {
		players[i] = &lobbyv1.Player{
			PlayerId:   p.PlayerId,
			PlayerName: p.PlayerName,
			Ready:      p.Ready,
		}
	}

	return &lobbyv1.RoomResponse{
		RoomId:     r.id,
		Status:     r.status,
		OwnerId:    r.ownerID,
		Players:    players,
		MaxPlayers: r.maxPlayers,
		JoinUrl:    fmt.Sprintf("/v1/rooms/%s/join", r.id),
	}
}

func requireTrimmed(value, field string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", status.Errorf(codes.InvalidArgument, "%s is required", field)
	}
	return trimmed, nil
}
