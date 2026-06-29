package gateway

import "testing"

func TestConfigDefaults(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.HTTPAddr != ":8080" {
		t.Fatalf("HTTPAddr = %q, want :8080", cfg.HTTPAddr)
	}
	if cfg.GameGRPCAddr != "localhost:50051" {
		t.Fatalf("GameGRPCAddr = %q, want localhost:50051", cfg.GameGRPCAddr)
	}
	if cfg.LobbyGRPCAddr != "localhost:50052" {
		t.Fatalf("LobbyGRPCAddr = %q, want localhost:50052", cfg.LobbyGRPCAddr)
	}
	if cfg.LobbyBackupGRPCAddr != "" {
		t.Fatalf("LobbyBackupGRPCAddr = %q, want empty", cfg.LobbyBackupGRPCAddr)
	}
	if cfg.LobbyBackupPromoteURL != "" {
		t.Fatalf("LobbyBackupPromoteURL = %q, want empty", cfg.LobbyBackupPromoteURL)
	}
}

func TestConfigFromEnvUsesContainerServiceAddress(t *testing.T) {
	t.Setenv("HTTP_ADDR", ":9090")
	t.Setenv("GAME_GRPC_ADDR", "game:50051")
	t.Setenv("LOBBY_GRPC_ADDR", "lobby-primary:50052")
	t.Setenv("LOBBY_BACKUP_GRPC_ADDR", "lobby-backup:50052")
	t.Setenv("LOBBY_BACKUP_PROMOTE_URL", "http://lobby-backup:8081/replication/promote")

	cfg := ConfigFromEnv()

	if cfg.HTTPAddr != ":9090" {
		t.Fatalf("HTTPAddr = %q, want :9090", cfg.HTTPAddr)
	}
	if cfg.GameGRPCAddr != "game:50051" {
		t.Fatalf("GameGRPCAddr = %q, want game:50051", cfg.GameGRPCAddr)
	}
	if cfg.LobbyGRPCAddr != "lobby-primary:50052" {
		t.Fatalf("LobbyGRPCAddr = %q, want lobby-primary:50052", cfg.LobbyGRPCAddr)
	}
	if cfg.LobbyBackupGRPCAddr != "lobby-backup:50052" {
		t.Fatalf("LobbyBackupGRPCAddr = %q, want lobby-backup:50052", cfg.LobbyBackupGRPCAddr)
	}
	if cfg.LobbyBackupPromoteURL != "http://lobby-backup:8081/replication/promote" {
		t.Fatalf("LobbyBackupPromoteURL = %q, want promotion URL", cfg.LobbyBackupPromoteURL)
	}
}
