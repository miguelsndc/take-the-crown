package game

import (
	"math"
	"sort"
)

const (
	MaxPlayers   = 8
	MinPlayers   = 2
	PlayerRadius = 15.0
	CrownRadius  = 10.0

	PLAYER_SPEED         = 200.0
	CROWN_TARGET_TIME    = 20.0
	CROWN_TRANSFER_DELAY = 0.75
	ROUND_RESTART_DELAY  = 4.0
)

const (
	RoundWaiting  = "waiting"
	RoundPlaying  = "playing"
	RoundFinished = "finished"
)

type Vec2 struct {
	X float64
	Y float64
}

type CrownSnapshot struct {
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	HolderID string  `json:"holder_id"`
}

type Crown struct {
	Position          Vec2
	CooldownRemaining float64
	HolderID          string
}

type Input struct {
	Up    bool
	Down  bool
	Left  bool
	Right bool
}

type Player struct {
	ID             string
	Position       Vec2
	CrownTime      float64
	Input          Input
	Inventory      [2]string
	SpeedRemaining float64
	GhostRemaining float64
}

type World struct {
	width   float64
	height  float64
	players map[string]*Player
	crown   Crown

	status           string
	winnerID         string
	restartRemaining float64

	pickups              []Pickup
	nextPickupID         uint64
	pickupSpawnRemaining float64
}

type Snapshot struct {
	Players          []Player
	Crown            CrownSnapshot
	Pickups          []Pickup
	Status           string
	WinnerID         string
	RestartRemaining float64
	TargetTime       float64
}

func NewWorld(width, height float64) *World {
	if width <= 0 || height <= 0 {
		panic("world dimensions must be positive")
	}

	world := &World{
		width:   width,
		height:  height,
		players: make(map[string]*Player),
		status:  RoundWaiting,
	}
	world.resetCrown()
	return world
}

func (w *World) AddPlayer(id string) bool {
	if _, exists := w.players[id]; exists {
		return false
	}
	if len(w.players) >= MaxPlayers {
		return false
	}

	w.players[id] = &Player{
		ID:       id,
		Position: w.spawnPosition(len(w.players), max(MinPlayers, len(w.players)+1)),
	}

	if len(w.players) >= MinPlayers && w.status == RoundWaiting {
		w.startRound()
	}
	return true
}

func (w *World) RemovePlayer(id string) bool {
	if _, exists := w.players[id]; !exists {
		return false
	}

	delete(w.players, id)
	if w.crown.HolderID == id {
		w.resetCrown()
	}

	if len(w.players) < MinPlayers {
		w.status = RoundWaiting
		w.winnerID = ""
		w.restartRemaining = 0
		w.pickups = nil
		w.resetCrown()
		for _, jogador := range w.players {
			jogador.CrownTime = 0
			jogador.Inventory = [2]string{}
		}
	}
	return true
}

func (w *World) SetInput(id string, input Input) bool {
	player, exists := w.players[id]
	if !exists {
		return false
	}

	player.Input = input
	return true
}

func (w *World) Player(id string) (Player, bool) {
	player, exists := w.players[id]
	if !exists {
		return Player{}, false
	}

	return *player, true
}

func (w *World) Update(dt float64) {
	if dt <= 0 {
		return
	}

	for _, jogador := range w.players {
		w.updateEffects(jogador, dt)
		w.movePlayer(jogador, dt)
	}

	switch w.status {
	case RoundPlaying:
		w.updatePickups(dt)
		w.updateCrown(dt)
	case RoundFinished:
		w.restartRemaining -= dt
		if w.restartRemaining <= 0 {
			if len(w.players) >= MinPlayers {
				w.startRound()
			} else {
				w.status = RoundWaiting
			}
		}
	}
}

func (w *World) movePlayer(player *Player, dt float64) {
	dx := 0.0
	dy := 0.0
	if player.Input.Up {
		dy--
	}
	if player.Input.Down {
		dy++
	}
	if player.Input.Left {
		dx--
	}
	if player.Input.Right {
		dx++
	}

	length := math.Hypot(dx, dy)
	if length > 0 {
		dx /= length
		dy /= length
	}

	speed := PLAYER_SPEED
	if player.SpeedRemaining > 0 {
		speed *= SPEED_MULTIPLIER
	}

	player.Position.X += dx * speed * dt
	player.Position.Y += dy * speed * dt
	player.Position.X = min(max(player.Position.X, PlayerRadius), w.width-PlayerRadius)
	player.Position.Y = min(max(player.Position.Y, PlayerRadius), w.height-PlayerRadius)
}

func (w *World) updateEffects(player *Player, dt float64) {
	player.SpeedRemaining = max(0, player.SpeedRemaining-dt)
	player.GhostRemaining = max(0, player.GhostRemaining-dt)
}

func (w *World) updateCrown(dt float64) {
	w.crown.CooldownRemaining = max(0, w.crown.CooldownRemaining-dt)

	if w.crown.HolderID == "" {
		for _, jogador := range w.orderedPlayers() {
			if circlesOverlap(
				jogador.Position,
				PlayerRadius,
				w.crown.Position,
				CrownRadius,
			) {
				w.crown.HolderID = jogador.ID
				w.crown.CooldownRemaining = CROWN_TRANSFER_DELAY
				return
			}
		}
		return
	}

	holder, exists := w.players[w.crown.HolderID]
	if !exists {
		w.resetCrown()
		return
	}

	holder.CrownTime += dt
	if holder.CrownTime >= CROWN_TARGET_TIME {
		holder.CrownTime = CROWN_TARGET_TIME
		w.status = RoundFinished
		w.winnerID = holder.ID
		w.restartRemaining = ROUND_RESTART_DELAY
		return
	}

	if w.crown.CooldownRemaining > 0 || holder.GhostRemaining > 0 {
		return
	}

	for _, jogador := range w.orderedPlayers() {
		if jogador.ID == holder.ID || jogador.GhostRemaining > 0 {
			continue
		}
		if circlesOverlap(jogador.Position, PlayerRadius, holder.Position, PlayerRadius) {
			w.crown.HolderID = jogador.ID
			w.crown.CooldownRemaining = CROWN_TRANSFER_DELAY
			return
		}
	}
}

func (w *World) startRound() {
	w.status = RoundPlaying
	w.winnerID = ""
	w.restartRemaining = 0
	w.resetCrown()

	players := w.orderedPlayers()
	for index, jogador := range players {
		jogador.Position = w.spawnPosition(index, len(players))
		jogador.CrownTime = 0
		jogador.Input = Input{}
		jogador.Inventory = [2]string{}
		jogador.SpeedRemaining = 0
		jogador.GhostRemaining = 0
	}
	w.resetPickups()
}

func (w *World) resetCrown() {
	w.crown = Crown{
		Position: Vec2{
			X: w.width / 2,
			Y: w.height / 2,
		},
	}
}

func (w *World) spawnPosition(index, total int) Vec2 {
	distanciaMaxima := math.Min(w.width, w.height)/2 - PlayerRadius - 10
	distancia := math.Min(280, max(0, distanciaMaxima))
	angle := 2 * math.Pi * float64(index) / float64(max(1, total))
	return Vec2{
		X: w.width/2 + math.Cos(angle)*distancia,
		Y: w.height/2 + math.Sin(angle)*distancia,
	}
}

func (w *World) orderedPlayers() []*Player {
	players := make([]*Player, 0, len(w.players))
	for _, player := range w.players {
		players = append(players, player)
	}
	sort.Slice(players, func(i, j int) bool {
		return players[i].ID < players[j].ID
	})
	return players
}

func (w *World) CrownSnapshot() CrownSnapshot {
	position := w.crown.Position
	if holder, exists := w.players[w.crown.HolderID]; exists {
		position = holder.Position
	}
	return CrownSnapshot{
		X:        position.X,
		Y:        position.Y,
		HolderID: w.crown.HolderID,
	}
}

func (w *World) Snapshot() Snapshot {
	players := make([]Player, 0, len(w.players))
	for _, player := range w.orderedPlayers() {
		players = append(players, *player)
	}

	pickups := make([]Pickup, len(w.pickups))
	copy(pickups, w.pickups)

	return Snapshot{
		Players:          players,
		Crown:            w.CrownSnapshot(),
		Pickups:          pickups,
		Status:           w.status,
		WinnerID:         w.winnerID,
		RestartRemaining: max(0, w.restartRemaining),
		TargetTime:       CROWN_TARGET_TIME,
	}
}

func circlesOverlap(p1 Vec2, r1 float64, p2 Vec2, r2 float64) bool {
	dx := p1.X - p2.X
	dy := p1.Y - p2.Y
	radiusSum := r1 + r2
	return dx*dx+dy*dy <= radiusSum*radiusSum
}
