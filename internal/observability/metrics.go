package observability

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	GatewayWebSocketSessions = promauto.NewCounter(prometheus.CounterOpts{
		Name: "voxel_gateway_websocket_sessions_total",
		Help: "Total de sessoes WebSocket aceitas pelo Gateway.",
	})
	GatewayActiveWebSockets = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "voxel_gateway_active_websockets",
		Help: "Sessoes WebSocket atualmente abertas no Gateway.",
	})
	GatewayWebSocketMessages = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "voxel_gateway_websocket_messages_total",
		Help: "Mensagens WebSocket processadas pelo Gateway por direcao.",
	}, []string{"direction"})
	GatewayWebSocketBytes = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "voxel_gateway_websocket_bytes_total",
		Help: "Bytes WebSocket processados pelo Gateway por direcao.",
	}, []string{"direction"})
	GatewayPushInputDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "voxel_gateway_push_input_duration_seconds",
		Help:    "Latencia da chamada Gateway -> Game PushInput.",
		Buckets: prometheus.DefBuckets,
	})
	GatewayRealtimeErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "voxel_gateway_realtime_errors_total",
		Help: "Erros no pipeline WebSocket/gRPC do Gateway.",
	}, []string{"operation"})

	GamePushInputs = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "voxel_game_push_inputs_total",
		Help: "Inputs recebidos pelo Game via PushInput por resultado.",
	}, []string{"result"})
	GameWatchStreams = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "voxel_game_watch_streams_total",
		Help: "Streams WatchMatch iniciados pelo Game por resultado final.",
	}, []string{"result"})
	GameActiveWatchStreams = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "voxel_game_active_watch_streams",
		Help: "Streams WatchMatch atualmente ativos no Game.",
	})
	GameActiveMatches = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "voxel_game_active_matches",
		Help: "Partidas em memoria no Game.",
	})
	GameActiveSubscribers = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "voxel_game_active_subscribers",
		Help: "Assinantes de snapshots atualmente conectados ao Game.",
	})
	GamePlayers = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "voxel_game_players",
		Help: "Jogadores registrados nas partidas em memoria do Game.",
	})
	GameTicks = promauto.NewCounter(prometheus.CounterOpts{
		Name: "voxel_game_ticks_total",
		Help: "Ticks autoritativos processados pelo relogio do Game.",
	})
	GameTickDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "voxel_game_tick_duration_seconds",
		Help:    "Duracao do processamento de um tick autoritativo do Game.",
		Buckets: []float64{0.0005, 0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1},
	})
	GameSnapshotDrops = promauto.NewCounter(prometheus.CounterOpts{
		Name: "voxel_game_snapshot_drops_total",
		Help: "Snapshots descartados por buffer cheio de assinante lento.",
	})

	LobbyRooms = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "voxel_lobby_rooms",
		Help: "Salas atualmente mantidas pelo Lobby.",
	})
	LobbyPlayers = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "voxel_lobby_players",
		Help: "Jogadores atualmente registrados em salas do Lobby.",
	})
	LobbyRoomEvents = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "voxel_lobby_room_events_total",
		Help: "Eventos de sala processados pelo Lobby.",
	}, []string{"event", "result"})
)

func MetricsHandler() http.Handler {
	return promhttp.Handler()
}
