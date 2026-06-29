package game

import (
	"context"
	"testing"

	matchv1 "voxel-royale/gen/match"
)

func TestServerStreamMatchReturnsAuthoritativeSnapshot(t *testing.T) {
	server := NewServer()

	state, err := server.StreamMatch(context.Background(), &matchv1.PlayerInput{
		PlayerId:      "player-1",
		MoveX:         1.5,
		MoveY:         -2,
		InputSequence: 1,
	})
	if err != nil {
		t.Fatalf("StreamMatch returned error: %v", err)
	}

	if state.Tick != 1 {
		t.Fatalf("Tick = %d, want 1", state.Tick)
	}
	if len(state.Players) != 1 {
		t.Fatalf("players length = %d, want 1", len(state.Players))
	}
	if len(state.Chests) != len(chestTemplates) {
		t.Fatalf("chests length = %d, want %d", len(state.Chests), len(chestTemplates))
	}
	if state.SafeZone == nil {
		t.Fatal("SafeZone should be present")
	}
	if state.MatchEnded {
		t.Fatal("match should not end with a single player at tick 1")
	}

	player := state.Players[0]
	if player.PlayerId != "player-1" {
		t.Fatalf("player id = %q, want player-1", player.PlayerId)
	}
	if player.X != 1.5 || player.Y != -2 {
		t.Fatalf("player coordinates = (%v,%v), want (1.5,-2)", player.X, player.Y)
	}
	if !player.IsAlive {
		t.Fatal("player should be alive")
	}
	if player.Health != maxHealth {
		t.Fatalf("player health = %d, want %d", player.Health, maxHealth)
	}
	if player.Weapon != weaponPistol {
		t.Fatalf("player weapon = %q, want %q", player.Weapon, weaponPistol)
	}
	if len(state.Ranking) != 1 || state.Ranking[0].PlayerId != "player-1" || state.Ranking[0].Place != 1 {
		t.Fatalf("ranking = %+v, want player-1 in first place", state.Ranking)
	}
}

func TestServerStreamMatchRejectsMissingPlayerID(t *testing.T) {
	server := NewServer()

	if _, err := server.StreamMatch(context.Background(), &matchv1.PlayerInput{}); err == nil {
		t.Fatal("StreamMatch returned nil error for missing player_id")
	}
}

func TestStartMatchRequiresRoomID(t *testing.T) {
	server := NewServer()

	if _, err := server.StartMatch(context.Background(), &matchv1.StartMatchRequest{}); err == nil {
		t.Fatal("StartMatch returned nil error for missing room_id")
	}
}

func TestStartMatchPreSpawnsRoster(t *testing.T) {
	server := NewServer()

	resp, err := server.StartMatch(context.Background(), &matchv1.StartMatchRequest{
		RoomId: "room-1",
		Players: []*matchv1.MatchPlayer{
			{PlayerId: "player-room-1-1", PlayerName: "Ana"},
			{PlayerId: "player-room-1-2", PlayerName: "Bruno"},
		},
	})
	if err != nil {
		t.Fatalf("StartMatch failed: %v", err)
	}
	if resp.MatchId != "room-1" || !resp.Started {
		t.Fatalf("response = %+v, want match_id room-1 and started true", resp)
	}

	// Um jogador da sala faz stream e ja deve ver os dois jogadores posicionados.
	state, err := server.StreamMatch(context.Background(), &matchv1.PlayerInput{
		PlayerId: "player-room-1-1",
		RoomId:   "room-1",
	})
	if err != nil {
		t.Fatalf("StreamMatch failed: %v", err)
	}
	if len(state.Players) != 2 {
		t.Fatalf("players length = %d, want 2 (roster pre-spawned)", len(state.Players))
	}

	first := findPlayer(t, state, "player-room-1-1")
	second := findPlayer(t, state, "player-room-1-2")
	spawnDistance := distance(vec2{first.X, first.Y}, vec2{second.X, second.Y})
	if spawnDistance < 50 {
		t.Fatalf("spawn distance = %v, want roster players more spread out", spawnDistance)
	}
}

func TestStreamMatchIsolatesRoomsByRoomID(t *testing.T) {
	server := NewServer()
	ctx := context.Background()

	if _, err := server.StreamMatch(ctx, &matchv1.PlayerInput{PlayerId: "a", RoomId: "room-1"}); err != nil {
		t.Fatalf("StreamMatch room-1 failed: %v", err)
	}
	if _, err := server.StreamMatch(ctx, &matchv1.PlayerInput{PlayerId: "b", RoomId: "room-2"}); err != nil {
		t.Fatalf("StreamMatch room-2 failed: %v", err)
	}

	state1, err := server.StreamMatch(ctx, &matchv1.PlayerInput{PlayerId: "a", RoomId: "room-1"})
	if err != nil {
		t.Fatalf("StreamMatch room-1 (2) failed: %v", err)
	}
	if len(state1.Players) != 1 || state1.Players[0].PlayerId != "a" {
		t.Fatalf("room-1 players = %+v, want only player a", state1.Players)
	}
}

func TestStartMatchResetsExistingRoomMatch(t *testing.T) {
	server := NewServer()
	ctx := context.Background()

	if _, err := server.StreamMatch(ctx, &matchv1.PlayerInput{PlayerId: "ghost", RoomId: "room-9"}); err != nil {
		t.Fatalf("StreamMatch failed: %v", err)
	}

	if _, err := server.StartMatch(ctx, &matchv1.StartMatchRequest{
		RoomId:  "room-9",
		Players: []*matchv1.MatchPlayer{{PlayerId: "player-room-9-1"}},
	}); err != nil {
		t.Fatalf("StartMatch failed: %v", err)
	}

	state, err := server.StreamMatch(ctx, &matchv1.PlayerInput{PlayerId: "player-room-9-1", RoomId: "room-9"})
	if err != nil {
		t.Fatalf("StreamMatch failed: %v", err)
	}
	for _, p := range state.Players {
		if p.PlayerId == "ghost" {
			t.Fatal("StartMatch should reset the room match, dropping stale players")
		}
	}
}

func TestServerStreamMatchClampsMovementAndIgnoresStaleInput(t *testing.T) {
	server := NewServer()

	state, err := server.StreamMatch(context.Background(), &matchv1.PlayerInput{
		PlayerId:      "runner",
		MoveX:         100,
		InputSequence: 1,
	})
	if err != nil {
		t.Fatalf("StreamMatch returned error: %v", err)
	}

	player := findPlayer(t, state, "runner")
	if player.X != maxMovePerTick || player.Y != 0 {
		t.Fatalf("player coordinates = (%v,%v), want (%v,0)", player.X, player.Y, maxMovePerTick)
	}

	state, err = server.StreamMatch(context.Background(), &matchv1.PlayerInput{
		PlayerId:      "runner",
		MoveX:         100,
		InputSequence: 1,
	})
	if err != nil {
		t.Fatalf("StreamMatch returned error: %v", err)
	}

	player = findPlayer(t, state, "runner")
	if player.X != maxMovePerTick || player.Y != 0 {
		t.Fatalf("stale input moved player to (%v,%v), want (%v,0)", player.X, player.Y, maxMovePerTick)
	}
}

func TestServerStreamMatchBlocksMovementIntoRock(t *testing.T) {
	server := NewServer()
	rock := rockObstacles[0]
	start := vec2{x: rock.pos.x - rock.radius - playerCollisionRadius - 0.75, y: rock.pos.y}

	if _, err := server.StreamMatch(context.Background(), &matchv1.PlayerInput{
		PlayerId:      "blocked",
		InputSequence: 1,
	}); err != nil {
		t.Fatalf("join failed: %v", err)
	}

	server.mu.Lock()
	server.matches[globalMatchKey].players["blocked"].pos = start
	server.mu.Unlock()

	state, err := server.StreamMatch(context.Background(), &matchv1.PlayerInput{
		PlayerId:      "blocked",
		MoveX:         maxMovePerTick,
		InputSequence: 2,
	})
	if err != nil {
		t.Fatalf("StreamMatch returned error: %v", err)
	}

	player := findPlayer(t, state, "blocked")
	pos := vec2{x: player.X, y: player.Y}
	if hasRockCollision(pos, playerCollisionRadius) {
		t.Fatalf("player moved inside rock hitbox at (%v,%v)", player.X, player.Y)
	}
	if player.X >= rock.pos.x-rock.radius-playerCollisionRadius {
		t.Fatalf("player crossed rock boundary: X = %v", player.X)
	}
}

func TestServerStreamMatchOpensChestAndEquipsWeapon(t *testing.T) {
	server := NewServer()

	state, err := server.StreamMatch(context.Background(), &matchv1.PlayerInput{
		PlayerId:      "looter",
		MoveX:         1,
		OpenChest:     true,
		InputSequence: 1,
	})
	if err != nil {
		t.Fatalf("StreamMatch returned error: %v", err)
	}

	player := findPlayer(t, state, "looter")
	if player.Weapon != weaponRifle {
		t.Fatalf("player weapon = %q, want %q", player.Weapon, weaponRifle)
	}

	chest := findChest(t, state, "chest-01")
	if !chest.IsOpened {
		t.Fatal("chest-01 should be opened")
	}
	if chest.OpenedByPlayerId != "looter" {
		t.Fatalf("chest opened by = %q, want looter", chest.OpenedByPlayerId)
	}
}

func TestServerStreamMatchRockBlocksBullets(t *testing.T) {
	server := NewServer()
	rock := rockObstacles[0]

	if _, err := server.StreamMatch(context.Background(), &matchv1.PlayerInput{PlayerId: "attacker", InputSequence: 1}); err != nil {
		t.Fatalf("attacker join failed: %v", err)
	}
	if _, err := server.StreamMatch(context.Background(), &matchv1.PlayerInput{PlayerId: "target", InputSequence: 1}); err != nil {
		t.Fatalf("target join failed: %v", err)
	}

	server.mu.Lock()
	match := server.matches[globalMatchKey]
	match.players["attacker"].weapon = weaponRifle
	match.players["attacker"].pos = vec2{rock.pos.x - 6, rock.pos.y}
	match.players["target"].pos = vec2{rock.pos.x + 6, rock.pos.y}
	server.mu.Unlock()

	state, err := server.StreamMatch(context.Background(), &matchv1.PlayerInput{
		PlayerId:       "attacker",
		IsAttacking:    true,
		TargetPlayerId: "target",
		InputSequence:  2,
	})
	if err != nil {
		t.Fatalf("attack failed: %v", err)
	}

	attacker := findPlayer(t, state, "attacker")
	target := findPlayer(t, state, "target")
	if attacker.DamageDealt != 0 {
		t.Fatalf("damage dealt through rock = %d, want 0", attacker.DamageDealt)
	}
	if target.Health != maxHealth || target.DamageTaken != 0 {
		t.Fatalf("target after blocked shot = health %d damageTaken %d, want %d/0", target.Health, target.DamageTaken, maxHealth)
	}
}

func TestServerStreamMatchAccountsDamageEliminationAndRanking(t *testing.T) {
	server := NewServer()

	if _, err := server.StreamMatch(context.Background(), &matchv1.PlayerInput{PlayerId: "attacker", InputSequence: 1}); err != nil {
		t.Fatalf("attacker join failed: %v", err)
	}
	if _, err := server.StreamMatch(context.Background(), &matchv1.PlayerInput{PlayerId: "target", InputSequence: 1}); err != nil {
		t.Fatalf("target join failed: %v", err)
	}

	server.mu.Lock()
	server.matches[globalMatchKey].players["attacker"].pos = vec2{0, 0}
	server.matches[globalMatchKey].players["target"].pos = vec2{6, 0}
	server.mu.Unlock()

	var state *matchv1.GameState
	for sequence := int64(2); sequence <= 7; sequence++ {
		var err error
		state, err = server.StreamMatch(context.Background(), &matchv1.PlayerInput{
			PlayerId:       "attacker",
			IsAttacking:    true,
			TargetPlayerId: "target",
			InputSequence:  sequence,
		})
		if err != nil {
			t.Fatalf("attack %d failed: %v", sequence, err)
		}
	}

	attacker := findPlayer(t, state, "attacker")
	target := findPlayer(t, state, "target")
	if attacker.DamageDealt != maxHealth {
		t.Fatalf("damage dealt = %d, want %d", attacker.DamageDealt, maxHealth)
	}
	if attacker.Eliminations != 1 {
		t.Fatalf("eliminations = %d, want 1", attacker.Eliminations)
	}
	if target.IsAlive {
		t.Fatal("target should be eliminated")
	}
	if target.Health != 0 {
		t.Fatalf("target health = %d, want 0", target.Health)
	}
	if target.DamageTaken != maxHealth {
		t.Fatalf("target damage taken = %d, want %d", target.DamageTaken, maxHealth)
	}
	if !state.MatchEnded {
		t.Fatal("match should end when one of two players remains")
	}
	if state.Ranking[0].PlayerId != "attacker" || state.Ranking[0].Place != 1 {
		t.Fatalf("first ranking entry = %+v, want attacker in first place", state.Ranking[0])
	}
	if state.Ranking[1].PlayerId != "target" || state.Ranking[1].Place != 2 {
		t.Fatalf("second ranking entry = %+v, want target in second place", state.Ranking[1])
	}
}

func TestServerStreamMatchShotgunDamagesCloseTarget(t *testing.T) {
	server := NewServer()

	if _, err := server.StreamMatch(context.Background(), &matchv1.PlayerInput{PlayerId: "attacker", InputSequence: 1}); err != nil {
		t.Fatalf("attacker join failed: %v", err)
	}
	if _, err := server.StreamMatch(context.Background(), &matchv1.PlayerInput{PlayerId: "target", InputSequence: 1}); err != nil {
		t.Fatalf("target join failed: %v", err)
	}

	server.mu.Lock()
	match := server.matches[globalMatchKey]
	match.players["attacker"].weapon = weaponShotgun
	match.players["attacker"].pos = vec2{0, 0}
	match.players["target"].pos = vec2{6, 0}
	server.mu.Unlock()

	state, err := server.StreamMatch(context.Background(), &matchv1.PlayerInput{
		PlayerId:       "attacker",
		IsAttacking:    true,
		TargetPlayerId: "target",
		InputSequence:  2,
	})
	if err != nil {
		t.Fatalf("shotgun attack failed: %v", err)
	}

	attacker := findPlayer(t, state, "attacker")
	target := findPlayer(t, state, "target")
	wantDamage := weaponProfiles[weaponShotgun].damage
	if attacker.DamageDealt != wantDamage {
		t.Fatalf("shotgun damage dealt = %d, want %d", attacker.DamageDealt, wantDamage)
	}
	if target.Health != maxHealth-wantDamage {
		t.Fatalf("target health = %d, want %d", target.Health, maxHealth-wantDamage)
	}
	if target.DamageTaken != wantDamage {
		t.Fatalf("target damage taken = %d, want %d", target.DamageTaken, wantDamage)
	}
}

func TestServerStreamMatchAppliesSafeZoneDamage(t *testing.T) {
	server := NewServer()

	if _, err := server.StreamMatch(context.Background(), &matchv1.PlayerInput{PlayerId: "wanderer", InputSequence: 1}); err != nil {
		t.Fatalf("join failed: %v", err)
	}

	server.mu.Lock()
	server.matches[globalMatchKey].players["wanderer"].pos = vec2{arenaHalfSize, arenaHalfSize}
	server.mu.Unlock()

	state, err := server.StreamMatch(context.Background(), &matchv1.PlayerInput{
		PlayerId:      "wanderer",
		InputSequence: 2,
	})
	if err != nil {
		t.Fatalf("StreamMatch returned error: %v", err)
	}

	player := findPlayer(t, state, "wanderer")
	if player.Health != maxHealth-safeZoneDamage {
		t.Fatalf("player health = %d, want %d", player.Health, maxHealth-safeZoneDamage)
	}
	if player.DamageTaken != safeZoneDamage {
		t.Fatalf("damage taken = %d, want %d", player.DamageTaken, safeZoneDamage)
	}
}

func TestServerStreamMatchSpawnsThreeWeaponTypesInChests(t *testing.T) {
	server := NewServer()

	state, err := server.StreamMatch(context.Background(), &matchv1.PlayerInput{
		PlayerId:      "scout",
		InputSequence: 1,
	})
	if err != nil {
		t.Fatalf("StreamMatch returned error: %v", err)
	}

	weapons := map[string]bool{}
	for _, chest := range state.Chests {
		weapons[chest.Weapon] = true
	}
	for _, weapon := range []string{weaponPistol, weaponRifle, weaponShotgun} {
		if !weapons[weapon] {
			t.Fatalf("weapon %q was not spawned in chests: %+v", weapon, weapons)
		}
	}
}

func findPlayer(t *testing.T, state *matchv1.GameState, playerID string) *matchv1.PlayerSnapshot {
	t.Helper()
	for _, player := range state.Players {
		if player.PlayerId == playerID {
			return player
		}
	}
	t.Fatalf("player %q not found in %+v", playerID, state.Players)
	return nil
}

func findChest(t *testing.T, state *matchv1.GameState, chestID string) *matchv1.ChestSnapshot {
	t.Helper()
	for _, chest := range state.Chests {
		if chest.ChestId == chestID {
			return chest
		}
	}
	t.Fatalf("chest %q not found in %+v", chestID, state.Chests)
	return nil
}
