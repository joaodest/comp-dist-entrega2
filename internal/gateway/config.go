package gateway

import "os"

type Config struct {
	HTTPAddr              string
	GameGRPCAddr          string
	LobbyGRPCAddr         string
	LobbyBackupGRPCAddr   string
	LobbyBackupPromoteURL string
}

func DefaultConfig() Config {
	return Config{
		HTTPAddr:      ":8080",
		GameGRPCAddr:  "localhost:50051",
		LobbyGRPCAddr: "localhost:50052",
	}
}

func ConfigFromEnv() Config {
	cfg := DefaultConfig()

	if value := os.Getenv("HTTP_ADDR"); value != "" {
		cfg.HTTPAddr = value
	}
	if value := os.Getenv("GAME_GRPC_ADDR"); value != "" {
		cfg.GameGRPCAddr = value
	}
	if value := os.Getenv("LOBBY_GRPC_ADDR"); value != "" {
		cfg.LobbyGRPCAddr = value
	}
	if value := os.Getenv("LOBBY_BACKUP_GRPC_ADDR"); value != "" {
		cfg.LobbyBackupGRPCAddr = value
	}
	if value := os.Getenv("LOBBY_BACKUP_PROMOTE_URL"); value != "" {
		cfg.LobbyBackupPromoteURL = value
	}

	return cfg
}
