package gateway

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	matchv1 "voxel-royale/gen/match"
	"voxel-royale/internal/observability"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/encoding/protojson"
)

// realtimeBridge liga o WebSocket do navegador ao Game via gRPC: encaminha os
// inputs do cliente (PushInput) e distribui os snapshots do relogio do servidor
// (WatchMatch) de volta ao WebSocket. E o ponto de "fan-out" do Gateway.
type realtimeBridge struct {
	game matchv1.GameServiceClient
}

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 4096,
	// Origem liberada: o cliente e servido pelo proprio dev server (proxy Vite)
	// ou pelo mesmo host em producao. Restringir na fase de deploy/hardening.
	CheckOrigin: func(_ *http.Request) bool { return true },
}

// jsonSnapshot espelha a serializacao do grpc-gateway (camelCase, int64 como
// string), para o cliente WebSocket consumir o mesmo formato dos web services.
var jsonSnapshot = protojson.MarshalOptions{UseProtoNames: false, EmitUnpopulated: true}

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = 25 * time.Second
)

// handleMatchWS atende GET /v1/match/ws?room=<id>&player=<id>. Mantem a conexao
// durante toda a partida (NETW-01), recebendo inputs (NETW-02), encaminhando-os
// ao Game (NETW-03) e reenviando os snapshots assinados (NETW-04).
func (b *realtimeBridge) handleMatchWS(w http.ResponseWriter, r *http.Request) {
	playerID := strings.TrimSpace(r.URL.Query().Get("player"))
	if playerID == "" {
		observability.GatewayRealtimeErrors.WithLabelValues("missing_player").Inc()
		http.Error(w, "query param 'player' is required", http.StatusBadRequest)
		return
	}
	roomID := strings.TrimSpace(r.URL.Query().Get("room"))

	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		observability.GatewayRealtimeErrors.WithLabelValues("upgrade").Inc()
		return // o upgrader ja respondeu o erro HTTP
	}
	defer func() { _ = conn.Close() }()
	observability.GatewayWebSocketSessions.Inc()
	observability.GatewayActiveWebSockets.Inc()
	defer observability.GatewayActiveWebSockets.Dec()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	stream, err := b.game.WatchMatch(ctx, &matchv1.WatchMatchRequest{RoomId: roomID, PlayerId: playerID})
	if err != nil {
		observability.GatewayRealtimeErrors.WithLabelValues("watch_match").Inc()
		log.Printf("gateway ws: WatchMatch failed (room=%q player=%q): %v", roomID, playerID, err)
		return
	}

	// Game -> WebSocket: repassa cada snapshot do relogio do servidor ao cliente.
	go b.pumpSnapshots(ctx, cancel, conn, stream)

	// WebSocket -> Game: cada input do cliente vira um PushInput gRPC.
	b.pumpInputs(ctx, conn, roomID, playerID)
}

// pumpSnapshots repassa os snapshots do stream gRPC para o WebSocket e envia
// pings periodicos para manter a conexao viva.
func (b *realtimeBridge) pumpSnapshots(
	ctx context.Context,
	cancel context.CancelFunc,
	conn *websocket.Conn,
	stream matchv1.GameService_WatchMatchClient,
) {
	defer cancel()

	ping := time.NewTicker(pingPeriod)
	defer ping.Stop()

	snapshots := make(chan []byte, 16)
	go func() {
		defer close(snapshots)
		for {
			snapshot, err := stream.Recv()
			if err != nil {
				if ctx.Err() == nil {
					observability.GatewayRealtimeErrors.WithLabelValues("snapshot_recv").Inc()
				}
				return
			}
			data, err := jsonSnapshot.Marshal(snapshot)
			if err != nil {
				observability.GatewayRealtimeErrors.WithLabelValues("snapshot_marshal").Inc()
				continue
			}
			select {
			case snapshots <- data:
			case <-ctx.Done():
				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ping.C:
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case data, ok := <-snapshots:
			if !ok {
				return
			}
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				observability.GatewayRealtimeErrors.WithLabelValues("snapshot_write").Inc()
				return
			}
			observability.GatewayWebSocketMessages.WithLabelValues("out").Inc()
			observability.GatewayWebSocketBytes.WithLabelValues("out").Add(float64(len(data)))
		}
	}
}

// pumpInputs le inputs do WebSocket e os encaminha ao Game. Tambem detecta a
// desconexao do cliente (ReadMessage retorna erro), encerrando a sessao.
func (b *realtimeBridge) pumpInputs(ctx context.Context, conn *websocket.Conn, roomID, playerID string) {
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		observability.GatewayWebSocketMessages.WithLabelValues("in").Inc()
		observability.GatewayWebSocketBytes.WithLabelValues("in").Add(float64(len(data)))

		var input matchv1.PlayerInput
		if err := protojson.Unmarshal(data, &input); err != nil {
			observability.GatewayRealtimeErrors.WithLabelValues("input_unmarshal").Inc()
			continue // ignora frames malformados sem derrubar a sessao
		}
		// A identidade e a sala vem da conexao (autoridade do Gateway), nunca do
		// payload do cliente.
		input.PlayerId = playerID
		input.RoomId = roomID

		start := time.Now()
		if _, err := b.game.PushInput(ctx, &input); err != nil {
			if ctx.Err() != nil {
				return
			}
			observability.GatewayRealtimeErrors.WithLabelValues("push_input").Inc()
			log.Printf("gateway ws: PushInput failed (room=%q player=%q): %v", roomID, playerID, err)
		}
		observability.GatewayPushInputDuration.Observe(time.Since(start).Seconds())
	}
}
