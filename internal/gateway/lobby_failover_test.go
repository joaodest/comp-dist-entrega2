package gateway

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	lobbyv1 "voxel-royale/gen/lobby"
	lobbyserver "voxel-royale/internal/lobby"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestLobbyFailoverPromotesBackupWhenPrimaryUnavailable(t *testing.T) {
	primaryAddr := closedTCPAddr(t)
	backup := lobbyserver.NewServer().WithReplication(lobbyserver.ReplicationBackup, nil)
	backupAddr, stopBackup := startTestLobbyGRPC(t, backup)
	defer stopBackup()

	promoteServer := httptest.NewServer(http.HandlerFunc(backup.HandlePromoteHTTP))
	defer promoteServer.Close()

	proxy, err := newLobbyFailoverProxy(
		primaryAddr,
		backupAddr,
		promoteServer.URL,
		[]grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())},
	)
	if err != nil {
		t.Fatalf("newLobbyFailoverProxy failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/rooms", strings.NewReader(`{"ownerName":"Ana","maxPlayers":4}`))
	recorder := httptest.NewRecorder()

	proxy.handleHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var resp lobbyv1.RoomResponse
	if err := protojson.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response unmarshal failed: %v", err)
	}
	if resp.RoomId == "" {
		t.Fatal("expected room to be created through promoted backup")
	}

	got, err := backup.GetRoom(context.Background(), &lobbyv1.GetRoomRequest{RoomId: resp.RoomId})
	if err != nil {
		t.Fatalf("backup GetRoom failed after failover: %v", err)
	}
	if got.OwnerId != resp.OwnerId {
		t.Fatalf("backup owner = %q, response owner = %q", got.OwnerId, resp.OwnerId)
	}
	if proxy.activeEndpoint().name != "backup" {
		t.Fatalf("active endpoint = %q, want backup", proxy.activeEndpoint().name)
	}
}

func startTestLobbyGRPC(t *testing.T, srv lobbyv1.LobbyServiceServer) (string, func()) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	grpcServer := grpc.NewServer()
	lobbyv1.RegisterLobbyServiceServer(grpcServer, srv)
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = grpcServer.Serve(listener)
	}()
	return listener.Addr().String(), func() {
		grpcServer.Stop()
		<-done
	}
}

func closedTCPAddr(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("close listener failed: %v", err)
	}
	return addr
}
