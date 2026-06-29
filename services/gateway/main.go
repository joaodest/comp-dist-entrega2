package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"voxel-royale/internal/gateway"
	"voxel-royale/internal/observability"
)

func main() {
	cfg := gateway.ConfigFromEnv()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	shutdownTracing, err := observability.SetupTracing(ctx, "voxel-gateway")
	if err != nil {
		log.Printf("gateway tracing disabled: %v", err)
	} else {
		defer func() { _ = shutdownTracing(context.Background()) }()
	}

	handler, err := gateway.NewProxyMux(ctx, cfg)
	if err != nil {
		log.Fatalf("gateway setup failed: %v", err)
	}

	server := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: observability.HTTPHandler("gateway-http", handler),
	}

	go func() {
		<-ctx.Done()
		_ = server.Shutdown(context.Background())
	}()

	log.Printf(
		"gateway http listening on %s, proxying game at %s and lobby primary at %s (backup %s)",
		cfg.HTTPAddr,
		cfg.GameGRPCAddr,
		cfg.LobbyGRPCAddr,
		cfg.LobbyBackupGRPCAddr,
	)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("gateway http server failed: %v", err)
	}
}
