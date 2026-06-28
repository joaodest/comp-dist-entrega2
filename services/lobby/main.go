package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	lobbyv1 "voxel-royale/gen/lobby"
	matchv1 "voxel-royale/gen/match"
	"voxel-royale/internal/lobby"
	"voxel-royale/internal/observability"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	grpcAddr := env("GRPC_ADDR", ":50052")
	healthAddr := env("HEALTH_ADDR", ":8081")
	gameAddr := env("GAME_GRPC_ADDR", "localhost:50051")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	shutdownTracing, err := observability.SetupTracing(ctx, "voxel-lobby")
	if err != nil {
		log.Printf("lobby tracing disabled: %v", err)
	} else {
		defer func() { _ = shutdownTracing(context.Background()) }()
	}

	gameConn, err := grpc.NewClient(
		gameAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		observability.GRPCClientOption(),
	)
	if err != nil {
		log.Fatalf("lobby could not create game client: %v", err)
	}
	defer func() { _ = gameConn.Close() }()

	starter := &gameMatchStarter{client: matchv1.NewGameServiceClient(gameConn)}
	lobbyServer := lobby.NewServer().WithMatchStarter(starter)

	grpcServer := grpc.NewServer(observability.GRPCServerOption())
	lobbyv1.RegisterLobbyServiceServer(grpcServer, lobbyServer)

	listener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("lobby grpc listen failed: %v", err)
	}

	ready := &atomic.Bool{}
	health := healthServer(healthAddr, ready, gameAddr)
	go func() {
		log.Printf("lobby health listening on %s", healthAddr)
		if err := health.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("lobby health server failed: %v", err)
		}
	}()

	go func() {
		<-ctx.Done()
		grpcServer.GracefulStop()
		_ = health.Shutdown(context.Background())
	}()

	log.Printf("lobby grpc listening on %s (game at %s)", grpcAddr, gameAddr)
	ready.Store(true)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("lobby grpc server failed: %v", err)
	}
}

// gameMatchStarter adapta o cliente gRPC gerado do Game para a interface
// lobby.MatchStarter, mantendo o pacote de dominio livre do contrato gerado.
type gameMatchStarter struct {
	client matchv1.GameServiceClient
}

func (g *gameMatchStarter) StartMatch(ctx context.Context, roomID string, maxPlayers int32, players []lobby.RosterPlayer) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	req := &matchv1.StartMatchRequest{
		RoomId:     roomID,
		MaxPlayers: maxPlayers,
		Players:    make([]*matchv1.MatchPlayer, 0, len(players)),
	}
	for _, p := range players {
		req.Players = append(req.Players, &matchv1.MatchPlayer{
			PlayerId:   p.PlayerID,
			PlayerName: p.PlayerName,
		})
	}
	_, err := g.client.StartMatch(ctx, req)
	return err
}

func healthServer(addr string, ready *atomic.Bool, gameAddr string) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if !ready.Load() {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), time.Second)
		defer cancel()
		var dialer net.Dialer
		conn, err := dialer.DialContext(ctx, "tcp", gameAddr)
		if err != nil {
			http.Error(w, "game dependency not ready", http.StatusServiceUnavailable)
			return
		}
		_ = conn.Close()
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.Handle("/metrics", observability.MetricsHandler())
	return &http.Server{Addr: addr, Handler: observability.HTTPHandler("lobby-health", mux)}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
