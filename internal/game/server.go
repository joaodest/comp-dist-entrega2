package game

import (
	"context"
	"math"
	"sort"
	"strings"
	"sync"

	matchv1 "voxel-royale/gen/match"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	arenaHalfSize        = float32(50)
	maxMovePerTick       = float32(2.5)
	chestOpenRange       = float32(2.25)
	initialSafeZoneRange = float32(45)
	finalSafeZoneRange   = float32(5)
	maxMatchTicks        = int64(300)
	safeZoneDamage       = int32(8)
	maxHealth            = int32(100)

	weaponPistol  = "pistol"
	weaponRifle   = "rifle"
	weaponShotgun = "shotgun"
)

var weaponProfiles = map[string]weaponProfile{
	weaponPistol:  {damage: 18, rangeUnits: 10, cooldownTicks: 1},
	weaponRifle:   {damage: 24, rangeUnits: 16, cooldownTicks: 1},
	weaponShotgun: {damage: 42, rangeUnits: 5, cooldownTicks: 2},
}

var spawnPoints = []vec2{
	{0, 0},
	{6, 0},
	{-6, 0},
	{0, 6},
	{0, -6},
	{8, 8},
	{-8, 8},
	{8, -8},
	{-8, -8},
	{14, 0},
	{-14, 0},
	{0, 14},
	{0, -14},
}

var chestTemplates = []chestState{
	{id: "chest-01", pos: vec2{3, 0}, weapon: weaponRifle},
	{id: "chest-02", pos: vec2{-3, 0}, weapon: weaponShotgun},
	{id: "chest-03", pos: vec2{0, 3}, weapon: weaponPistol},
	{id: "chest-04", pos: vec2{10, 10}, weapon: weaponRifle},
	{id: "chest-05", pos: vec2{-10, 10}, weapon: weaponShotgun},
	{id: "chest-06", pos: vec2{10, -10}, weapon: weaponPistol},
	{id: "chest-07", pos: vec2{-10, -10}, weapon: weaponRifle},
	{id: "chest-08", pos: vec2{18, 0}, weapon: weaponShotgun},
	{id: "chest-09", pos: vec2{0, -18}, weapon: weaponPistol},
}

type Server struct {
	matchv1.UnimplementedGameServiceServer

	mu    sync.Mutex
	match *matchState
}

type matchState struct {
	tick       int64
	players    map[string]*playerState
	playerIDs  []string
	chests     map[string]*chestState
	chestIDs   []string
	matchEnded bool
}

type playerState struct {
	id                string
	pos               vec2
	health            int32
	weapon            string
	alive             bool
	eliminations      int32
	damageDealt       int32
	damageTaken       int32
	joinedTick        int64
	survivedTicks     int64
	lastInputSequence int64
	lastAttackTick    int64
}

type chestState struct {
	id       string
	pos      vec2
	weapon   string
	opened   bool
	openedBy string
}

type weaponProfile struct {
	damage        int32
	rangeUnits    float32
	cooldownTicks int64
}

type vec2 struct {
	x float32
	y float32
}

func NewServer() *Server {
	return &Server{match: newMatchState()}
}

func (s *Server) StreamMatch(_ context.Context, input *matchv1.PlayerInput) (*matchv1.GameState, error) {
	if input == nil || strings.TrimSpace(input.PlayerId) == "" {
		return nil, status.Error(codes.InvalidArgument, "player_id is required")
	}
	if !isFinite(input.MoveX) || !isFinite(input.MoveY) || !isFinite(input.AimX) || !isFinite(input.AimY) {
		return nil, status.Error(codes.InvalidArgument, "movement and aim values must be finite")
	}

	playerID := strings.TrimSpace(input.PlayerId)

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.match == nil {
		s.match = newMatchState()
	}

	match := s.match
	player := match.ensurePlayer(playerID)
	match.tick++

	if !match.matchEnded && player.alive && !isStaleInput(player, input) {
		applyInputSequence(player, input)
		match.movePlayer(player, input.MoveX, input.MoveY)
		if input.OpenChest {
			match.openNearestChest(player)
		}
		if input.IsAttacking {
			match.attack(player, input)
		}
	}

	match.applySafeZoneDamage()
	match.refreshSurvivalTicks()
	match.updateMatchEnd()

	return match.snapshot(), nil
}

func newMatchState() *matchState {
	match := &matchState{
		players: make(map[string]*playerState),
		chests:  make(map[string]*chestState),
	}
	for i := range chestTemplates {
		chest := chestTemplates[i]
		match.chests[chest.id] = &chest
		match.chestIDs = append(match.chestIDs, chest.id)
	}
	return match
}

func (m *matchState) ensurePlayer(playerID string) *playerState {
	if player, ok := m.players[playerID]; ok {
		return player
	}

	player := &playerState{
		id:             playerID,
		pos:            spawnFor(len(m.playerIDs)),
		health:         maxHealth,
		weapon:         weaponPistol,
		alive:          true,
		joinedTick:     m.tick,
		lastAttackTick: -1,
	}
	m.players[playerID] = player
	m.playerIDs = append(m.playerIDs, playerID)
	return player
}

func (m *matchState) movePlayer(player *playerState, dx, dy float32) {
	move := clampVector(vec2{dx, dy}, maxMovePerTick)
	player.pos.x = clamp(player.pos.x+move.x, -arenaHalfSize, arenaHalfSize)
	player.pos.y = clamp(player.pos.y+move.y, -arenaHalfSize, arenaHalfSize)
}

func (m *matchState) openNearestChest(player *playerState) {
	var nearest *chestState
	nearestDistance := float32(math.MaxFloat32)
	for _, chestID := range m.chestIDs {
		chest := m.chests[chestID]
		if chest.opened {
			continue
		}
		distance := distance(player.pos, chest.pos)
		if distance <= chestOpenRange && distance < nearestDistance {
			nearest = chest
			nearestDistance = distance
		}
	}
	if nearest == nil {
		return
	}
	nearest.opened = true
	nearest.openedBy = player.id
	player.weapon = nearest.weapon
}

func (m *matchState) attack(attacker *playerState, input *matchv1.PlayerInput) {
	profile := profileFor(attacker.weapon)
	if attacker.lastAttackTick >= 0 && m.tick-attacker.lastAttackTick < profile.cooldownTicks {
		return
	}

	target := m.findAttackTarget(attacker, input, profile.rangeUnits)
	if target == nil {
		return
	}

	damage := minInt32(profile.damage, target.health)
	target.health -= damage
	target.damageTaken += damage
	attacker.damageDealt += damage
	attacker.lastAttackTick = m.tick

	if target.health <= 0 {
		target.health = 0
		target.alive = false
		target.survivedTicks = m.tick - target.joinedTick
		attacker.eliminations++
	}
}

func (m *matchState) findAttackTarget(attacker *playerState, input *matchv1.PlayerInput, rangeUnits float32) *playerState {
	if targetID := strings.TrimSpace(input.TargetPlayerId); targetID != "" {
		target := m.players[targetID]
		if target != nil && target.alive && target.id != attacker.id && distance(attacker.pos, target.pos) <= rangeUnits {
			return target
		}
		return nil
	}

	aim := normalized(vec2{input.AimX, input.AimY})
	var nearest *playerState
	nearestDistance := float32(math.MaxFloat32)
	for _, playerID := range m.playerIDs {
		candidate := m.players[playerID]
		if candidate == nil || !candidate.alive || candidate.id == attacker.id {
			continue
		}
		offset := vec2{candidate.pos.x - attacker.pos.x, candidate.pos.y - attacker.pos.y}
		distanceToTarget := length(offset)
		if distanceToTarget > rangeUnits || distanceToTarget >= nearestDistance {
			continue
		}
		if aim != (vec2{}) && dot(aim, normalized(offset)) < 0.5 {
			continue
		}
		nearest = candidate
		nearestDistance = distanceToTarget
	}
	return nearest
}

func (m *matchState) applySafeZoneDamage() {
	zone := safeZoneAtTick(m.tick)
	for _, playerID := range m.playerIDs {
		player := m.players[playerID]
		if player == nil || !player.alive || distance(player.pos, vec2{zone.CenterX, zone.CenterY}) <= zone.Radius {
			continue
		}
		damage := minInt32(safeZoneDamage, player.health)
		player.health -= damage
		player.damageTaken += damage
		if player.health <= 0 {
			player.health = 0
			player.alive = false
			player.survivedTicks = m.tick - player.joinedTick
		}
	}
}

func (m *matchState) refreshSurvivalTicks() {
	for _, playerID := range m.playerIDs {
		player := m.players[playerID]
		if player != nil && player.alive {
			player.survivedTicks = m.tick - player.joinedTick
		}
	}
}

func (m *matchState) updateMatchEnd() {
	if m.tick >= maxMatchTicks {
		m.matchEnded = true
		return
	}
	if len(m.playerIDs) < 2 {
		return
	}
	alive := 0
	for _, playerID := range m.playerIDs {
		if player := m.players[playerID]; player != nil && player.alive {
			alive++
		}
	}
	m.matchEnded = alive <= 1
}

func (m *matchState) snapshot() *matchv1.GameState {
	remainingTicks := maxMatchTicks - m.tick
	if remainingTicks < 0 {
		remainingTicks = 0
	}

	return &matchv1.GameState{
		Tick:           m.tick,
		Players:        m.playerSnapshots(),
		Chests:         m.chestSnapshots(),
		SafeZone:       safeZoneAtTick(m.tick),
		Ranking:        m.ranking(),
		MatchEnded:     m.matchEnded,
		RemainingTicks: remainingTicks,
	}
}

func (m *matchState) playerSnapshots() []*matchv1.PlayerSnapshot {
	players := make([]*matchv1.PlayerSnapshot, 0, len(m.playerIDs))
	for _, playerID := range m.playerIDs {
		player := m.players[playerID]
		if player == nil {
			continue
		}
		players = append(players, &matchv1.PlayerSnapshot{
			PlayerId:      player.id,
			X:             player.pos.x,
			Y:             player.pos.y,
			IsAlive:       player.alive,
			Health:        player.health,
			Weapon:        player.weapon,
			Eliminations:  player.eliminations,
			DamageDealt:   player.damageDealt,
			DamageTaken:   player.damageTaken,
			SurvivedTicks: player.survivedTicks,
		})
	}
	return players
}

func (m *matchState) chestSnapshots() []*matchv1.ChestSnapshot {
	chests := make([]*matchv1.ChestSnapshot, 0, len(m.chestIDs))
	for _, chestID := range m.chestIDs {
		chest := m.chests[chestID]
		if chest == nil {
			continue
		}
		chests = append(chests, &matchv1.ChestSnapshot{
			ChestId:          chest.id,
			X:                chest.pos.x,
			Y:                chest.pos.y,
			IsOpened:         chest.opened,
			Weapon:           chest.weapon,
			OpenedByPlayerId: chest.openedBy,
		})
	}
	return chests
}

func (m *matchState) ranking() []*matchv1.RankingEntry {
	entries := make([]*matchv1.RankingEntry, 0, len(m.playerIDs))
	for _, playerID := range m.playerIDs {
		player := m.players[playerID]
		if player == nil {
			continue
		}
		entries = append(entries, &matchv1.RankingEntry{
			PlayerId:      player.id,
			IsAlive:       player.alive,
			Health:        player.health,
			Eliminations:  player.eliminations,
			DamageDealt:   player.damageDealt,
			SurvivedTicks: player.survivedTicks,
		})
	}

	sort.SliceStable(entries, func(i, j int) bool {
		left := entries[i]
		right := entries[j]
		if left.IsAlive != right.IsAlive {
			return left.IsAlive
		}
		if left.IsAlive {
			if left.Eliminations != right.Eliminations {
				return left.Eliminations > right.Eliminations
			}
			if left.Health != right.Health {
				return left.Health > right.Health
			}
		} else if left.SurvivedTicks != right.SurvivedTicks {
			return left.SurvivedTicks > right.SurvivedTicks
		}
		if left.DamageDealt != right.DamageDealt {
			return left.DamageDealt > right.DamageDealt
		}
		if left.SurvivedTicks != right.SurvivedTicks {
			return left.SurvivedTicks > right.SurvivedTicks
		}
		return left.PlayerId < right.PlayerId
	})

	for i := range entries {
		entries[i].Place = int32(i + 1)
	}
	return entries
}

func safeZoneAtTick(tick int64) *matchv1.SafeZoneSnapshot {
	progress := float32(tick) / float32(maxMatchTicks)
	if progress > 1 {
		progress = 1
	}
	radius := initialSafeZoneRange - ((initialSafeZoneRange - finalSafeZoneRange) * progress)
	return &matchv1.SafeZoneSnapshot{
		CenterX: 0,
		CenterY: 0,
		Radius:  radius,
		Phase:   tick / (maxMatchTicks / 5),
	}
}

func spawnFor(index int) vec2 {
	if index < len(spawnPoints) {
		return spawnPoints[index]
	}
	angle := float64(index) * 2 * math.Pi / 50
	ring := float32(18 + (index/len(spawnPoints))*6)
	return vec2{x: ring * float32(math.Cos(angle)), y: ring * float32(math.Sin(angle))}
}

func isStaleInput(player *playerState, input *matchv1.PlayerInput) bool {
	return input.InputSequence > 0 && input.InputSequence <= player.lastInputSequence
}

func applyInputSequence(player *playerState, input *matchv1.PlayerInput) {
	if input.InputSequence > 0 {
		player.lastInputSequence = input.InputSequence
	}
}

func profileFor(weapon string) weaponProfile {
	if profile, ok := weaponProfiles[weapon]; ok {
		return profile
	}
	return weaponProfiles[weaponPistol]
}

func clampVector(v vec2, maxLength float32) vec2 {
	length := length(v)
	if length == 0 || length <= maxLength {
		return v
	}
	scale := maxLength / length
	return vec2{x: v.x * scale, y: v.y * scale}
}

func normalized(v vec2) vec2 {
	length := length(v)
	if length == 0 {
		return vec2{}
	}
	return vec2{x: v.x / length, y: v.y / length}
}

func distance(a, b vec2) float32 {
	return length(vec2{x: a.x - b.x, y: a.y - b.y})
}

func length(v vec2) float32 {
	return float32(math.Hypot(float64(v.x), float64(v.y)))
}

func dot(a, b vec2) float32 {
	return a.x*b.x + a.y*b.y
}

func clamp(value, minValue, maxValue float32) float32 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func minInt32(a, b int32) int32 {
	if a < b {
		return a
	}
	return b
}

func isFinite(value float32) bool {
	return !math.IsNaN(float64(value)) && !math.IsInf(float64(value), 0)
}
