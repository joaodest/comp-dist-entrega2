package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

type roomResponse struct {
	RoomID  string       `json:"roomId"`
	OwnerID string       `json:"ownerId"`
	Players []roomPlayer `json:"players"`
}

type roomPlayer struct {
	PlayerID string `json:"playerId"`
}

type playerInput struct {
	MoveX         float64 `json:"moveX"`
	MoveY         float64 `json:"moveY"`
	IsAttacking   bool    `json:"isAttacking"`
	InputSequence string  `json:"inputSequence"`
	OpenChest     bool    `json:"openChest"`
	AimX          float64 `json:"aimX"`
	AimY          float64 `json:"aimY"`
}

type stats struct {
	connected    atomic.Int64
	snapshots    atomic.Int64
	snapshotByte atomic.Int64
	inputs       atomic.Int64
	errors       atomic.Int64
}

func main() {
	gateway := flag.String("gateway", "http://localhost:8080", "URL base HTTP do Gateway")
	playerCount := flag.Int("players", 50, "quantidade de jogadores simulados")
	duration := flag.Duration("duration", 30*time.Second, "duracao da simulacao")
	sendEvery := flag.Duration("send-every", 66*time.Millisecond, "intervalo entre inputs por jogador")
	flag.Parse()

	if *playerCount < 1 {
		exitf("players must be >= 1")
	}

	ctx := context.Background()
	client := &http.Client{Timeout: 10 * time.Second}

	room, err := createRoom(ctx, client, *gateway, *playerCount)
	if err != nil {
		exitf("create room: %v", err)
	}

	playerIDs := []string{room.OwnerID}
	for i := 2; i <= *playerCount; i++ {
		joined, err := joinRoom(ctx, client, *gateway, room.RoomID, fmt.Sprintf("Bot %02d", i))
		if err != nil {
			exitf("join player %d: %v", i, err)
		}
		playerIDs = append(playerIDs, joined.Players[len(joined.Players)-1].PlayerID)
	}

	if err := startRoom(ctx, client, *gateway, room.RoomID, room.OwnerID); err != nil {
		exitf("start room: %v", err)
	}

	wsURL, err := websocketURL(*gateway, room.RoomID)
	if err != nil {
		exitf("websocket URL: %v", err)
	}

	runCtx, cancel := context.WithTimeout(ctx, *duration)
	defer cancel()

	var wg sync.WaitGroup
	var totals stats
	startedAt := time.Now()
	for i, playerID := range playerIDs {
		wg.Add(1)
		go runPlayer(runCtx, &wg, wsURL, playerID, i, *sendEvery, &totals)
	}
	wg.Wait()
	elapsed := time.Since(startedAt)

	result := map[string]any{
		"roomId":                  room.RoomID,
		"playersRequested":        *playerCount,
		"playersConnected":        totals.connected.Load(),
		"durationSeconds":         elapsed.Seconds(),
		"inputsSent":              totals.inputs.Load(),
		"snapshotsReceived":       totals.snapshots.Load(),
		"snapshotBytesReceived":   totals.snapshotByte.Load(),
		"snapshotsPerSecond":      float64(totals.snapshots.Load()) / math.Max(elapsed.Seconds(), 0.001),
		"avgSnapshotsPerPlayer":   float64(totals.snapshots.Load()) / math.Max(float64(*playerCount), 1),
		"simulatorErrors":         totals.errors.Load(),
		"prometheusMetrics":       fmt.Sprintf("%s/metrics", *gateway),
		"grafanaDashboard":        "http://localhost:3000/d/voxel-royale/voxel-royale",
		"jaegerSearch":            "http://localhost:16686/search",
		"recommendedPlayersGoal":  50,
		"recommendedDurationGoal": "30s or more",
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(result)
}

func createRoom(ctx context.Context, client *http.Client, gateway string, players int) (*roomResponse, error) {
	var out roomResponse
	err := postJSON(ctx, client, gateway+"/v1/rooms", map[string]any{
		"ownerName":  "Bot 01",
		"maxPlayers": players,
	}, &out)
	return &out, err
}

func joinRoom(ctx context.Context, client *http.Client, gateway, roomID, name string) (*roomResponse, error) {
	var out roomResponse
	err := postJSON(ctx, client, fmt.Sprintf("%s/v1/rooms/%s/join", gateway, url.PathEscape(roomID)), map[string]any{
		"playerName": name,
	}, &out)
	return &out, err
}

func startRoom(ctx context.Context, client *http.Client, gateway, roomID, ownerID string) error {
	return postJSON(ctx, client, fmt.Sprintf("%s/v1/rooms/%s/start", gateway, url.PathEscape(roomID)), map[string]any{
		"playerId": ownerID,
	}, &roomResponse{})
}

func postJSON(ctx context.Context, client *http.Client, endpoint string, in, out any) error {
	payload, err := json.Marshal(in)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func websocketURL(gateway, roomID string) (string, error) {
	u, err := url.Parse(gateway)
	if err != nil {
		return "", err
	}
	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	default:
		u.Scheme = "ws"
	}
	u.Path = "/v1/match/ws"
	q := u.Query()
	q.Set("room", roomID)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func runPlayer(ctx context.Context, wg *sync.WaitGroup, wsBase, playerID string, index int, sendEvery time.Duration, totals *stats) {
	defer wg.Done()

	u, err := url.Parse(wsBase)
	if err != nil {
		totals.errors.Add(1)
		return
	}
	q := u.Query()
	q.Set("player", playerID)
	u.RawQuery = q.Encode()

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, u.String(), nil)
	if err != nil {
		totals.errors.Add(1)
		return
	}
	totals.connected.Add(1)
	defer func() { _ = conn.Close() }()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				if ctx.Err() == nil {
					totals.errors.Add(1)
				}
				return
			}
			totals.snapshots.Add(1)
			totals.snapshotByte.Add(int64(len(data)))
		}
	}()

	rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(index)))
	ticker := time.NewTicker(sendEvery)
	defer ticker.Stop()

	var sequence int64
	for {
		select {
		case <-ctx.Done():
			_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "done"))
			<-done
			return
		case <-ticker.C:
			sequence++
			angle := rng.Float64() * 2 * math.Pi
			input := playerInput{
				MoveX:         math.Cos(angle),
				MoveY:         math.Sin(angle),
				IsAttacking:   sequence%12 == 0,
				InputSequence: fmt.Sprintf("%d", sequence),
				OpenChest:     sequence%45 == 0,
				AimX:          math.Cos(angle),
				AimY:          math.Sin(angle),
			}
			data, err := json.Marshal(input)
			if err != nil {
				totals.errors.Add(1)
				return
			}
			_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				totals.errors.Add(1)
				return
			}
			totals.inputs.Add(1)
		}
	}
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
