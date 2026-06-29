package lobby

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"

	lobbyv1 "voxel-royale/gen/lobby"
	"voxel-royale/internal/observability"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const defaultMaxPlayers = 50

// RosterPlayer e a visao minima que o Lobby passa ao Game ao iniciar a partida.
type RosterPlayer struct {
	PlayerID   string
	PlayerName string
}

// MatchStarter abstrai a chamada gRPC Lobby -> Game para iniciar a partida da
// sala. Mantida como interface para desacoplar o Lobby do cliente gerado e
// permitir testes com fakes.
type MatchStarter interface {
	StartMatch(ctx context.Context, roomID string, maxPlayers int32, players []RosterPlayer) error
}

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

	mu           sync.RWMutex
	rooms        map[string]*room
	nextID       uint64
	matchStarter MatchStarter

	replicationRole    ReplicationRole
	replicator         StateReplicator
	replicationVersion uint64
}

func NewServer() *Server {
	return &Server{
		rooms:           make(map[string]*room),
		replicationRole: ReplicationStandalone,
	}
}

// WithMatchStarter liga o Lobby ao Game. Sem um starter (ex.: testes), o inicio
// de sala apenas muda o status localmente, como na Fase 1.
func (s *Server) WithMatchStarter(starter MatchStarter) *Server {
	s.matchStarter = starter
	return s
}

func (s *Server) WithReplication(role ReplicationRole, replicator StateReplicator) *Server {
	s.replicationRole = role
	s.replicator = replicator
	return s
}

func (s *Server) ReplicationVersion() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.replicationVersion
}

// rosterOf captura os jogadores da sala para envio ao Game.
func rosterOf(r *room) []RosterPlayer {
	roster := make([]RosterPlayer, 0, len(r.players))
	for _, p := range r.players {
		roster = append(roster, RosterPlayer{PlayerID: p.PlayerId, PlayerName: p.PlayerName})
	}
	return roster
}

func (s *Server) ensureWritable() error {
	if s.replicationRole == ReplicationBackup {
		return status.Error(codes.FailedPrecondition, "backup lobby is read-only; writes must be sent to the primary")
	}
	return nil
}

func (s *Server) snapshotLocked() ReplicationSnapshot {
	roomIDs := make([]string, 0, len(s.rooms))
	for roomID := range s.rooms {
		roomIDs = append(roomIDs, roomID)
	}
	sort.Strings(roomIDs)

	snapshot := ReplicationSnapshot{
		NextID: s.nextID,
		Rooms:  make([]ReplicatedRoom, 0, len(roomIDs)),
	}
	for _, roomID := range roomIDs {
		r := s.rooms[roomID]
		if r == nil {
			continue
		}
		replicated := ReplicatedRoom{
			ID:           r.id,
			OwnerID:      r.ownerID,
			OwnerName:    r.ownerName,
			Status:       int32(r.status),
			MaxPlayers:   r.maxPlayers,
			NextPlayerID: r.nextPlayerID,
			Players:      make([]ReplicatedPlayer, 0, len(r.players)),
		}
		for _, p := range r.players {
			replicated.Players = append(replicated.Players, ReplicatedPlayer{
				PlayerID:   p.PlayerId,
				PlayerName: p.PlayerName,
				Ready:      p.Ready,
			})
		}
		snapshot.Rooms = append(snapshot.Rooms, replicated)
	}
	return snapshot
}

func (s *Server) restoreSnapshotLocked(snapshot ReplicationSnapshot, version uint64) {
	rooms := make(map[string]*room, len(snapshot.Rooms))
	for _, replicated := range snapshot.Rooms {
		players := make([]*lobbyv1.Player, 0, len(replicated.Players))
		playerSet := make(map[string]bool, len(replicated.Players))
		for _, p := range replicated.Players {
			players = append(players, &lobbyv1.Player{
				PlayerId:   p.PlayerID,
				PlayerName: p.PlayerName,
				Ready:      p.Ready,
			})
			playerSet[p.PlayerID] = true
		}
		rooms[replicated.ID] = &room{
			id:           replicated.ID,
			ownerID:      replicated.OwnerID,
			ownerName:    replicated.OwnerName,
			status:       lobbyv1.RoomStatus(replicated.Status),
			maxPlayers:   replicated.MaxPlayers,
			players:      players,
			playerSet:    playerSet,
			nextPlayerID: replicated.NextPlayerID,
		}
	}
	s.rooms = rooms
	s.nextID = snapshot.NextID
	s.replicationVersion = version
	s.observeStateLocked()
	observability.LobbyReplicationVersion.Set(float64(version))
}

func (s *Server) replicateCurrentLocked(ctx context.Context) error {
	if s.replicationRole != ReplicationPrimary || s.replicator == nil {
		return nil
	}

	s.replicationVersion++
	version := s.replicationVersion
	event := ReplicationEvent{
		Version:  version,
		Snapshot: s.snapshotLocked(),
	}
	if err := s.replicator.Replicate(ctx, event); err != nil {
		observability.LobbyReplicationEvents.WithLabelValues("send", "error").Inc()
		return status.Errorf(codes.Unavailable, "failed to replicate lobby state version %d: %v", version, err)
	}
	observability.LobbyReplicationEvents.WithLabelValues("send", "ok").Inc()
	observability.LobbyReplicationVersion.Set(float64(version))
	return nil
}

func (s *Server) commitReplicationLocked(ctx context.Context, previous ReplicationSnapshot, previousVersion uint64) error {
	if err := s.replicateCurrentLocked(ctx); err != nil {
		s.restoreSnapshotLocked(previous, previousVersion)
		return err
	}
	return nil
}

func (s *Server) ApplyReplicationEvent(_ context.Context, event ReplicationEvent) error {
	if event.Version == 0 {
		return status.Error(codes.InvalidArgument, "replication version must be greater than zero")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	switch {
	case event.Version == s.replicationVersion:
		if !reflect.DeepEqual(event.Snapshot, s.snapshotLocked()) {
			return status.Errorf(codes.FailedPrecondition, "replication version %d already applied with a different snapshot", event.Version)
		}
		observability.LobbyReplicationEvents.WithLabelValues("apply", "duplicate").Inc()
		return nil
	case event.Version != s.replicationVersion+1:
		return status.Errorf(codes.FailedPrecondition, "replication version gap: got %d, want %d", event.Version, s.replicationVersion+1)
	}

	s.restoreSnapshotLocked(event.Snapshot, event.Version)
	observability.LobbyReplicationEvents.WithLabelValues("apply", "ok").Inc()
	return nil
}

func (s *Server) CreateRoom(ctx context.Context, req *lobbyv1.CreateRoomRequest) (*lobbyv1.RoomResponse, error) {
	if err := s.ensureWritable(); err != nil {
		return nil, err
	}
	if req == nil || strings.TrimSpace(req.OwnerName) == "" {
		return nil, status.Error(codes.InvalidArgument, "owner_name is required")
	}

	maxPlayers := req.MaxPlayers
	if maxPlayers <= 0 {
		maxPlayers = defaultMaxPlayers
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	previous := s.snapshotLocked()
	previousVersion := s.replicationVersion

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
	if err := s.commitReplicationLocked(ctx, previous, previousVersion); err != nil {
		return nil, err
	}
	s.observeStateLocked()
	observability.LobbyRoomEvents.WithLabelValues("create", "ok").Inc()

	return roomToResponse(r), nil
}

func (s *Server) JoinRoom(ctx context.Context, req *lobbyv1.JoinRoomRequest) (*lobbyv1.RoomResponse, error) {
	if err := s.ensureWritable(); err != nil {
		return nil, err
	}
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

	previous := s.snapshotLocked()
	previousVersion := s.replicationVersion

	playerID := fmt.Sprintf("player-%s-%d", roomID, r.nextPlayerID)
	r.nextPlayerID++

	player := &lobbyv1.Player{
		PlayerId:   playerID,
		PlayerName: playerName,
	}
	r.players = append(r.players, player)
	r.playerSet[playerID] = true
	if err := s.commitReplicationLocked(ctx, previous, previousVersion); err != nil {
		return nil, err
	}
	s.observeStateLocked()
	observability.LobbyRoomEvents.WithLabelValues("join", "ok").Inc()

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

func (s *Server) StartRoom(ctx context.Context, req *lobbyv1.StartRoomRequest) (*lobbyv1.RoomResponse, error) {
	if err := s.ensureWritable(); err != nil {
		return nil, err
	}
	roomID, err := requireTrimmed(req.GetRoomId(), "room_id")
	if err != nil {
		return nil, err
	}
	playerID, err := requireTrimmed(req.GetPlayerId(), "player_id")
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	r, ok := s.rooms[roomID]
	if !ok {
		s.mu.Unlock()
		return nil, status.Errorf(codes.NotFound, "room %s not found", roomID)
	}
	if r.status != lobbyv1.RoomStatus_ROOM_STATUS_WAITING {
		s.mu.Unlock()
		return nil, status.Errorf(codes.FailedPrecondition, "room %s cannot be started (status: %v)", roomID, r.status)
	}
	if playerID != r.ownerID {
		s.mu.Unlock()
		return nil, status.Errorf(codes.PermissionDenied, "only the room owner can start the room")
	}

	previous := s.snapshotLocked()
	previousVersion := s.replicationVersion
	r.status = lobbyv1.RoomStatus_ROOM_STATUS_STARTED
	roster := rosterOf(r)
	maxPlayers := r.maxPlayers
	if err := s.commitReplicationLocked(ctx, previous, previousVersion); err != nil {
		s.mu.Unlock()
		observability.LobbyRoomEvents.WithLabelValues("start", "error").Inc()
		return nil, err
	}
	s.observeStateLocked()
	s.mu.Unlock()

	if err := s.triggerMatch(ctx, roomID, maxPlayers, roster); err != nil {
		if revertErr := s.revertToWaiting(ctx, roomID); revertErr != nil {
			observability.LobbyRoomEvents.WithLabelValues("start", "error").Inc()
			return nil, status.Errorf(codes.Internal, "failed to start match for room %s: %v; rollback replication failed: %v", roomID, err, revertErr)
		}
		observability.LobbyRoomEvents.WithLabelValues("start", "error").Inc()
		return nil, status.Errorf(codes.Internal, "failed to start match for room %s: %v", roomID, err)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok = s.rooms[roomID]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "room %s not found", roomID)
	}
	s.observeStateLocked()
	observability.LobbyRoomEvents.WithLabelValues("start", "ok").Inc()
	return roomToResponse(r), nil
}

// triggerMatch chama o Game (se houver starter configurado) fora do lock do
// Lobby, evitando segurar o mutex durante a chamada de rede.
func (s *Server) triggerMatch(ctx context.Context, roomID string, maxPlayers int32, roster []RosterPlayer) error {
	if s.matchStarter == nil {
		return nil
	}
	return s.matchStarter.StartMatch(ctx, roomID, maxPlayers, roster)
}

// revertToWaiting desfaz a transicao para STARTED quando o Game nao confirma o
// inicio da partida, deixando a sala utilizavel novamente.
func (s *Server) revertToWaiting(ctx context.Context, roomID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	previous := s.snapshotLocked()
	previousVersion := s.replicationVersion
	if r, ok := s.rooms[roomID]; ok && r.status == lobbyv1.RoomStatus_ROOM_STATUS_STARTED {
		r.status = lobbyv1.RoomStatus_ROOM_STATUS_WAITING
		if err := s.commitReplicationLocked(ctx, previous, previousVersion); err != nil {
			return err
		}
		s.observeStateLocked()
	}
	return nil
}

func (s *Server) LeaveRoom(ctx context.Context, req *lobbyv1.LeaveRoomRequest) (*lobbyv1.RoomResponse, error) {
	if err := s.ensureWritable(); err != nil {
		return nil, err
	}
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

	previous := s.snapshotLocked()
	previousVersion := s.replicationVersion

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

	if err := s.commitReplicationLocked(ctx, previous, previousVersion); err != nil {
		return nil, err
	}
	resp := roomToResponse(r)
	s.observeStateLocked()
	observability.LobbyRoomEvents.WithLabelValues("leave", "ok").Inc()
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

func (s *Server) SetReady(ctx context.Context, req *lobbyv1.SetReadyRequest) (*lobbyv1.RoomResponse, error) {
	if err := s.ensureWritable(); err != nil {
		return nil, err
	}
	roomID, err := requireTrimmed(req.GetRoomId(), "room_id")
	if err != nil {
		return nil, err
	}

	playerID, err := requireTrimmed(req.GetPlayerId(), "player_id")
	if err != nil {
		return nil, err
	}

	s.mu.Lock()

	r, ok := s.rooms[roomID]
	if !ok {
		s.mu.Unlock()
		return nil, status.Errorf(codes.NotFound, "room %s not found", roomID)
	}

	if r.status != lobbyv1.RoomStatus_ROOM_STATUS_WAITING {
		s.mu.Unlock()
		return nil, status.Errorf(
			codes.FailedPrecondition,
			"room %s is not in waiting state (status: %v)",
			roomID,
			r.status,
		)
	}

	if !r.playerSet[playerID] {
		s.mu.Unlock()
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
		s.mu.Unlock()
		return nil, status.Errorf(
			codes.NotFound,
			"player %s not found in room %s",
			playerID,
			roomID,
		)
	}

	previous := s.snapshotLocked()
	previousVersion := s.replicationVersion
	player.Ready = req.Ready

	autoStart := req.Ready && r.allReady()
	if autoStart {
		r.status = lobbyv1.RoomStatus_ROOM_STATUS_STARTED
	}
	roster := rosterOf(r)
	maxPlayers := r.maxPlayers
	if err := s.commitReplicationLocked(ctx, previous, previousVersion); err != nil {
		s.mu.Unlock()
		observability.LobbyRoomEvents.WithLabelValues("ready", "error").Inc()
		return nil, err
	}
	s.observeStateLocked()
	s.mu.Unlock()

	if autoStart {
		if err := s.triggerMatch(ctx, roomID, maxPlayers, roster); err != nil {
			if revertErr := s.revertToWaiting(ctx, roomID); revertErr != nil {
				observability.LobbyRoomEvents.WithLabelValues("ready", "error").Inc()
				return nil, status.Errorf(codes.Internal, "failed to start match for room %s: %v; rollback replication failed: %v", roomID, err, revertErr)
			}
			observability.LobbyRoomEvents.WithLabelValues("ready", "error").Inc()
			return nil, status.Errorf(codes.Internal, "failed to start match for room %s: %v", roomID, err)
		}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok = s.rooms[roomID]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "room %s not found", roomID)
	}
	s.observeStateLocked()
	if autoStart {
		observability.LobbyRoomEvents.WithLabelValues("ready_auto_start", "ok").Inc()
	} else {
		observability.LobbyRoomEvents.WithLabelValues("ready", "ok").Inc()
	}
	return roomToResponse(r), nil
}

func (s *Server) observeStateLocked() {
	var players int
	for _, room := range s.rooms {
		if room == nil {
			continue
		}
		players += len(room.players)
	}
	observability.LobbyRooms.Set(float64(len(s.rooms)))
	observability.LobbyPlayers.Set(float64(players))
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
