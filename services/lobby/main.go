package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	lobbyv1 "voxel-royale/gen/lobby"
	matchv1 "voxel-royale/gen/match"
	"voxel-royale/internal/lobby"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	grpcAddr := env("GRPC_ADDR", ":50052")
	healthAddr := env("HEALTH_ADDR", ":8081")
	gameAddr := env("GAME_GRPC_ADDR", "localhost:50051")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	gameConn, err := grpc.NewClient(gameAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("lobby could not create game client: %v", err)
	}
	defer func() { _ = gameConn.Close() }()

	starter := &gameMatchStarter{client: matchv1.NewGameServiceClient(gameConn)}
	lobbyServer := lobby.NewServer().WithMatchStarter(starter)

	grpcServer := grpc.NewServer()
	lobbyv1.RegisterLobbyServiceServer(grpcServer, lobbyServer)

	listener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("lobby grpc listen failed: %v", err)
	}

	health := healthServer(healthAddr)
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

func healthServer(addr string) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok\n"))
	})
	return &http.Server{Addr: addr, Handler: mux}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
