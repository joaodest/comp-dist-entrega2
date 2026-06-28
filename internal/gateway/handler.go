package gateway

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	lobbyv1 "voxel-royale/gen/lobby"
	matchv1 "voxel-royale/gen/match"
	"voxel-royale/internal/observability"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ReadinessCheck func(context.Context) error

func NewHealthMux(readinessChecks ...ReadinessCheck) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), time.Second)
		defer cancel()
		for _, check := range readinessChecks {
			if err := check(ctx); err != nil {
				http.Error(w, fmt.Sprintf("not ready: %v", err), http.StatusServiceUnavailable)
				return
			}
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.Handle("/metrics", observability.MetricsHandler())
	return mux
}

// NewProxyMux monta o roteador HTTP do Gateway: web services REST (grpc-gateway
// para Lobby e Game), o healthcheck e o endpoint WebSocket de tempo real
// (/v1/match/ws), que liga a sessao do navegador as RPCs gRPC PushInput e
// WatchMatch do Game (Fase 4).
func NewProxyMux(ctx context.Context, gameGRPCAddr, lobbyGRPCAddr string) (http.Handler, error) {
	proxy := runtime.NewServeMux()
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		observability.GRPCClientOption(),
	}

	if err := matchv1.RegisterGameServiceHandlerFromEndpoint(ctx, proxy, gameGRPCAddr, opts); err != nil {
		return nil, err
	}
	if err := lobbyv1.RegisterLobbyServiceHandlerFromEndpoint(ctx, proxy, lobbyGRPCAddr, opts); err != nil {
		return nil, err
	}

	// Conexao gRPC dedicada ao Game para o pipeline de tempo real (WebSocket).
	// E separada do proxy REST para manter o transporte realtime desacoplado.
	gameConn, err := grpc.NewClient(gameGRPCAddr, opts...)
	if err != nil {
		return nil, err
	}
	rt := &realtimeBridge{game: matchv1.NewGameServiceClient(gameConn)}

	mux := NewHealthMux(
		tcpReadinessCheck("game", gameGRPCAddr),
		tcpReadinessCheck("lobby", lobbyGRPCAddr),
	)
	mux.HandleFunc("/v1/match/ws", rt.handleMatchWS)
	mux.Handle("/", proxy)
	return mux, nil
}

func tcpReadinessCheck(name, addr string) ReadinessCheck {
	return func(ctx context.Context) error {
		var dialer net.Dialer
		conn, err := dialer.DialContext(ctx, "tcp", addr)
		if err != nil {
			return fmt.Errorf("%s grpc at %s unavailable: %w", name, addr, err)
		}
		_ = conn.Close()
		return nil
	}
}
