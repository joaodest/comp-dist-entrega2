package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	lobbyv1 "voxel-royale/gen/lobby"
	"voxel-royale/internal/observability"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const lobbyRPCTimeout = 5 * time.Second

type lobbyEndpoint struct {
	name   string
	addr   string
	client lobbyv1.LobbyServiceClient
}

type lobbyFailoverProxy struct {
	mu sync.RWMutex

	endpoints []lobbyEndpoint
	active    int

	promoteURL string
	httpClient *http.Client
}

var lobbyJSON = protojson.MarshalOptions{UseProtoNames: false, EmitUnpopulated: true}
var lobbyJSONUnmarshal = protojson.UnmarshalOptions{DiscardUnknown: true}

func newLobbyFailoverProxy(primaryAddr, backupAddr, promoteURL string, opts []grpc.DialOption) (*lobbyFailoverProxy, error) {
	endpoints := make([]lobbyEndpoint, 0, 2)
	for _, candidate := range []struct {
		name string
		addr string
	}{
		{name: "primary", addr: strings.TrimSpace(primaryAddr)},
		{name: "backup", addr: strings.TrimSpace(backupAddr)},
	} {
		if candidate.addr == "" {
			continue
		}
		conn, err := grpc.NewClient(candidate.addr, opts...)
		if err != nil {
			return nil, fmt.Errorf("create lobby %s client: %w", candidate.name, err)
		}
		endpoints = append(endpoints, lobbyEndpoint{
			name:   candidate.name,
			addr:   candidate.addr,
			client: lobbyv1.NewLobbyServiceClient(conn),
		})
	}

	if len(endpoints) == 0 {
		return nil, errors.New("at least one lobby endpoint is required")
	}

	return &lobbyFailoverProxy{
		endpoints:  endpoints,
		promoteURL: strings.TrimSpace(promoteURL),
		httpClient: &http.Client{Timeout: 2 * time.Second},
	}, nil
}

func (p *lobbyFailoverProxy) readinessCheck(ctx context.Context) error {
	active := p.activeEndpoint()
	if err := checkTCP(ctx, active.name, active.addr); err == nil {
		return nil
	}

	backup, ok := p.backupEndpoint()
	if !ok || backup.addr == active.addr {
		return checkTCP(ctx, active.name, active.addr)
	}
	if err := checkTCP(ctx, backup.name, backup.addr); err != nil {
		return fmt.Errorf("lobby primary and backup unavailable: %w", err)
	}
	return nil
}

func (p *lobbyFailoverProxy) handleHTTP(w http.ResponseWriter, r *http.Request) {
	resp, err := p.route(r)
	if err != nil {
		writeLobbyError(w, err)
		return
	}
	writeLobbyResponse(w, resp)
}

func (p *lobbyFailoverProxy) route(r *http.Request) (proto.Message, error) {
	ctx, cancel := context.WithTimeout(r.Context(), lobbyRPCTimeout)
	defer cancel()

	rest := strings.TrimPrefix(r.URL.Path, "/v1/rooms")
	if rest == "" || rest == "/" {
		if r.Method != http.MethodPost {
			return nil, status.Error(codes.Unimplemented, "method not allowed")
		}
		var req lobbyv1.CreateRoomRequest
		if err := decodeLobbyBody(r, &req); err != nil {
			return nil, err
		}
		return p.call(ctx, "CreateRoom", func(client lobbyv1.LobbyServiceClient) (proto.Message, error) {
			return client.CreateRoom(ctx, &req)
		})
	}

	parts := strings.Split(strings.Trim(rest, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		return nil, status.Error(codes.NotFound, "room path is required")
	}
	roomID, err := url.PathUnescape(parts[0])
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid room id: %v", err)
	}

	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			return nil, status.Error(codes.Unimplemented, "method not allowed")
		}
		req := &lobbyv1.GetRoomRequest{RoomId: roomID}
		return p.call(ctx, "GetRoom", func(client lobbyv1.LobbyServiceClient) (proto.Message, error) {
			return client.GetRoom(ctx, req)
		})
	}
	if len(parts) != 2 {
		return nil, status.Error(codes.NotFound, "unknown room route")
	}
	if r.Method != http.MethodPost {
		return nil, status.Error(codes.Unimplemented, "method not allowed")
	}

	switch parts[1] {
	case "join":
		var req lobbyv1.JoinRoomRequest
		if err := decodeLobbyBody(r, &req); err != nil {
			return nil, err
		}
		req.RoomId = roomID
		return p.call(ctx, "JoinRoom", func(client lobbyv1.LobbyServiceClient) (proto.Message, error) {
			return client.JoinRoom(ctx, &req)
		})
	case "start":
		var req lobbyv1.StartRoomRequest
		if err := decodeLobbyBody(r, &req); err != nil {
			return nil, err
		}
		req.RoomId = roomID
		return p.call(ctx, "StartRoom", func(client lobbyv1.LobbyServiceClient) (proto.Message, error) {
			return client.StartRoom(ctx, &req)
		})
	case "leave":
		var req lobbyv1.LeaveRoomRequest
		if err := decodeLobbyBody(r, &req); err != nil {
			return nil, err
		}
		req.RoomId = roomID
		return p.call(ctx, "LeaveRoom", func(client lobbyv1.LobbyServiceClient) (proto.Message, error) {
			return client.LeaveRoom(ctx, &req)
		})
	case "ready":
		var req lobbyv1.SetReadyRequest
		if err := decodeLobbyBody(r, &req); err != nil {
			return nil, err
		}
		req.RoomId = roomID
		return p.call(ctx, "SetReady", func(client lobbyv1.LobbyServiceClient) (proto.Message, error) {
			return client.SetReady(ctx, &req)
		})
	default:
		return nil, status.Error(codes.NotFound, "unknown room route")
	}
}

func (p *lobbyFailoverProxy) call(
	ctx context.Context,
	operation string,
	invoke func(lobbyv1.LobbyServiceClient) (proto.Message, error),
) (proto.Message, error) {
	endpoint := p.activeEndpoint()
	resp, err := invoke(endpoint.client)
	if err == nil || endpoint.name != "primary" || !shouldFailoverLobby(err) {
		return resp, err
	}

	if promoteErr := p.promoteBackup(ctx); promoteErr != nil {
		observability.GatewayLobbyFailovers.WithLabelValues(operation, "error").Inc()
		return nil, status.Errorf(codes.Unavailable, "lobby primary unavailable and backup promotion failed: %v", promoteErr)
	}

	backup := p.activeEndpoint()
	resp, err = invoke(backup.client)
	if err != nil {
		observability.GatewayLobbyFailovers.WithLabelValues(operation, "retry_error").Inc()
		return nil, err
	}
	observability.GatewayLobbyFailovers.WithLabelValues(operation, "ok").Inc()
	return resp, nil
}

func (p *lobbyFailoverProxy) activeEndpoint() lobbyEndpoint {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.endpoints[p.active]
}

func (p *lobbyFailoverProxy) backupEndpoint() (lobbyEndpoint, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, endpoint := range p.endpoints {
		if endpoint.name == "backup" {
			return endpoint, true
		}
	}
	return lobbyEndpoint{}, false
}

func (p *lobbyFailoverProxy) promoteBackup(ctx context.Context) error {
	p.mu.RLock()
	if p.endpoints[p.active].name == "backup" {
		p.mu.RUnlock()
		return nil
	}
	p.mu.RUnlock()

	backup, ok := p.backupEndpoint()
	if !ok {
		return errors.New("no lobby backup endpoint configured")
	}
	if p.promoteURL == "" {
		return errors.New("no lobby backup promotion URL configured")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.promoteURL, bytes.NewReader(nil))
	if err != nil {
		return fmt.Errorf("build promote request: %w", err)
	}
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("promote backup %s: %w", backup.addr, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("promote backup returned %s: %s", resp.Status, strings.TrimSpace(string(data)))
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	for idx, endpoint := range p.endpoints {
		if endpoint.name == "backup" {
			p.active = idx
			return nil
		}
	}
	return errors.New("backup endpoint disappeared during promotion")
}

func shouldFailoverLobby(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	switch status.Code(err) {
	case codes.Unavailable, codes.DeadlineExceeded:
		return true
	default:
		return false
	}
}

func decodeLobbyBody(r *http.Request, msg proto.Message) error {
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "read request body: %v", err)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		data = []byte("{}")
	}
	if err := lobbyJSONUnmarshal.Unmarshal(data, msg); err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid JSON body: %v", err)
	}
	return nil
}

func writeLobbyResponse(w http.ResponseWriter, msg proto.Message) {
	data, err := lobbyJSON.Marshal(msg)
	if err != nil {
		writeLobbyError(w, status.Errorf(codes.Internal, "marshal response: %v", err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(data)
	_, _ = w.Write([]byte("\n"))
}

func writeLobbyError(w http.ResponseWriter, err error) {
	code := status.Code(err)
	httpStatus := httpStatusFromGRPC(code)
	message := err.Error()
	if st, ok := status.FromError(err); ok {
		message = st.Message()
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"code":    code.String(),
		"message": message,
	})
}

func httpStatusFromGRPC(code codes.Code) int {
	switch code {
	case codes.OK:
		return http.StatusOK
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.NotFound:
		return http.StatusNotFound
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.FailedPrecondition:
		return http.StatusPreconditionFailed
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout
	case codes.Unimplemented:
		return http.StatusMethodNotAllowed
	default:
		return http.StatusInternalServerError
	}
}

func checkTCP(ctx context.Context, name, addr string) error {
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("%s grpc at %s unavailable: %w", name, addr, err)
	}
	_ = conn.Close()
	return nil
}
