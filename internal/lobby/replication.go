package lobby

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"voxel-royale/internal/observability"
)

type ReplicationRole string

const (
	ReplicationStandalone ReplicationRole = "standalone"
	ReplicationPrimary    ReplicationRole = "primary"
	ReplicationBackup     ReplicationRole = "backup"
)

func ParseReplicationRole(value string) ReplicationRole {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(ReplicationPrimary):
		return ReplicationPrimary
	case string(ReplicationBackup):
		return ReplicationBackup
	default:
		return ReplicationStandalone
	}
}

type StateReplicator interface {
	Replicate(ctx context.Context, event ReplicationEvent) error
}

type ReplicationEvent struct {
	Version  uint64              `json:"version"`
	Snapshot ReplicationSnapshot `json:"snapshot"`
}

type ReplicationSnapshot struct {
	NextID uint64           `json:"nextId"`
	Rooms  []ReplicatedRoom `json:"rooms"`
}

type ReplicatedRoom struct {
	ID           string             `json:"id"`
	OwnerID      string             `json:"ownerId"`
	OwnerName    string             `json:"ownerName"`
	Status       int32              `json:"status"`
	MaxPlayers   int32              `json:"maxPlayers"`
	NextPlayerID uint64             `json:"nextPlayerId"`
	Players      []ReplicatedPlayer `json:"players"`
}

type ReplicatedPlayer struct {
	PlayerID   string `json:"playerId"`
	PlayerName string `json:"playerName"`
	Ready      bool   `json:"ready"`
}

type HTTPReplicator struct {
	endpoint string
	timeout  time.Duration
	client   *http.Client
}

func NewHTTPReplicator(endpoint string, timeout time.Duration) *HTTPReplicator {
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	return &HTTPReplicator{
		endpoint: strings.TrimSpace(endpoint),
		timeout:  timeout,
		client:   &http.Client{Timeout: timeout},
	}
}

func (r *HTTPReplicator) Replicate(ctx context.Context, event ReplicationEvent) error {
	if r == nil || r.endpoint == "" {
		return nil
	}

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal replication event: %w", err)
	}

	reqCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, r.endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build replication request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("send replication event: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("replica returned %s: %s", resp.Status, strings.TrimSpace(string(data)))
	}
	return nil
}

func (s *Server) HandleReplicationHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var event ReplicationEvent
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := decoder.Decode(&event); err != nil {
		observability.LobbyReplicationEvents.WithLabelValues("apply", "bad_request").Inc()
		http.Error(w, fmt.Sprintf("invalid replication event: %v", err), http.StatusBadRequest)
		return
	}

	if err := s.ApplyReplicationEvent(r.Context(), event); err != nil {
		observability.LobbyReplicationEvents.WithLabelValues("apply", "error").Inc()
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("ok\n"))
}

func (s *Server) HandlePromoteHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.PromoteToPrimary()
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("promoted\n"))
}
