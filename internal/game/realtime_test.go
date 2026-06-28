package game

import (
	"context"
	"sync"
	"testing"
	"time"

	matchv1 "voxel-royale/gen/match"

	"google.golang.org/grpc"
)

// fakeWatchStream implementa matchv1.GameService_WatchMatchServer coletando os
// snapshots enviados, para validar o relogio do servidor sem rede.
type fakeWatchStream struct {
	grpc.ServerStream
	ctx  context.Context
	mu   sync.Mutex
	sent []*matchv1.GameState
}

func (f *fakeWatchStream) Context() context.Context { return f.ctx }

func (f *fakeWatchStream) Send(state *matchv1.GameState) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, state)
	return nil
}

func (f *fakeWatchStream) snapshots() []*matchv1.GameState {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*matchv1.GameState, len(f.sent))
	copy(out, f.sent)
	return out
}

func TestPushInputRequiresPlayerID(t *testing.T) {
	server := NewServer()
	if _, err := server.PushInput(context.Background(), &matchv1.PlayerInput{}); err == nil {
		t.Fatal("PushInput returned nil error for missing player_id")
	}
}

func TestRealtimeClockStreamsAdvancingSnapshots(t *testing.T) {
	server := NewServer()
	ctx, cancel := context.WithCancel(context.Background())
	stream := &fakeWatchStream{ctx: ctx}

	done := make(chan error, 1)
	go func() {
		done <- server.WatchMatch(&matchv1.WatchMatchRequest{RoomId: "room-rt", PlayerId: "p1"}, stream)
	}()

	// Buffer um input de movimento para a direita; o relogio deve consumi-lo em
	// um tick authoritative, desacoplado da chegada do pacote.
	if _, err := server.PushInput(ctx, &matchv1.PlayerInput{
		PlayerId:      "p1",
		RoomId:        "room-rt",
		MoveX:         1,
		InputSequence: 1,
	}); err != nil {
		t.Fatalf("PushInput failed: %v", err)
	}

	// Aguarda alguns ticks do relogio do servidor (tickHz) e encerra a sessao.
	time.Sleep(350 * time.Millisecond)
	cancel()
	<-done

	snapshots := stream.snapshots()
	if len(snapshots) < 2 {
		t.Fatalf("expected multiple streamed snapshots, got %d", len(snapshots))
	}

	first := snapshots[0]
	last := snapshots[len(snapshots)-1]
	if last.Tick <= first.Tick {
		t.Fatalf("clock did not advance: first tick %d, last tick %d", first.Tick, last.Tick)
	}

	var player *matchv1.PlayerSnapshot
	for _, p := range last.Players {
		if p.PlayerId == "p1" {
			player = p
		}
	}
	if player == nil {
		t.Fatalf("player p1 missing from snapshot: %+v", last.Players)
	}
	if player.X <= 0 {
		t.Fatalf("buffered input was not applied: X = %v, want > 0", player.X)
	}
}

func TestRealtimeAdvanceTickIgnoresStaleInput(t *testing.T) {
	match := newMatchState()
	player := match.ensurePlayer("p1")
	match.pendingInputs["p1"] = &matchv1.PlayerInput{
		PlayerId:      "p1",
		MoveX:         1,
		InputSequence: 2,
	}

	match.advanceTick()
	if player.pos.x != 1 {
		t.Fatalf("first input moved player to X = %v, want 1", player.pos.x)
	}

	match.pendingInputs["p1"] = &matchv1.PlayerInput{
		PlayerId:      "p1",
		MoveX:         1,
		InputSequence: 2,
	}
	match.advanceTick()
	if player.pos.x != 1 {
		t.Fatalf("stale input moved player to X = %v, want 1", player.pos.x)
	}
}

func TestRealtimeClockStopsWithoutSubscribers(t *testing.T) {
	server := NewServer()
	ctx, cancel := context.WithCancel(context.Background())
	stream := &fakeWatchStream{ctx: ctx}

	done := make(chan error, 1)
	go func() {
		done <- server.WatchMatch(&matchv1.WatchMatchRequest{RoomId: "room-stop", PlayerId: "p1"}, stream)
	}()

	time.Sleep(150 * time.Millisecond)
	cancel()
	<-done

	// Sem assinantes, o relogio deve se desligar (clockRunning volta a false).
	time.Sleep(150 * time.Millisecond)
	server.mu.Lock()
	match := server.matches["room-stop"]
	running := match != nil && match.clockRunning
	server.mu.Unlock()
	if running {
		t.Fatal("clock should stop after the last subscriber disconnects")
	}
}
