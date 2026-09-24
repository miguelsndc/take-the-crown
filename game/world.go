package game

import "math"

type Vec2 struct {
	X float64
	Y float64
}

type Input struct {
	Up    bool
	Down  bool
	Left  bool
	Right bool
}

type Player struct {
	ID       string
	Position Vec2
	Input    Input
}

type World struct {
	width   float64
	height  float64
	players map[string]*Player
}

type Snapshot struct {
	Players []Player `json:"player"`
}

const PLAYER_SPEED = 200.0

func NewWorld(width, height float64) *World {
	if width <= 0 || height <= 0 {
		panic("world dimensions must be positive")
	}

	return &World{
		width:   width,
		height:  height,
		players: make(map[string]*Player),
	}
}

func (w *World) AddPlayer(id string) bool {
	if _, exists := w.players[id]; exists {
		return false
	}
	w.players[id] = &Player{
		ID: id,
		Position: Vec2{
			X: w.width / 2,
			Y: w.height / 2,
		},
	}
	return true
}

func (w *World) RemovePlayer(id string) bool {
	if _, exists := w.players[id]; !exists {
		return false
	}
	delete(w.players, id)
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
	dx := 0.0
	dy := 0.0
	for _, player := range w.players {
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
		player.Position.X += dx * PLAYER_SPEED * dt
		player.Position.Y += dy * PLAYER_SPEED * dt
		player.Position.X = min(max(player.Position.X, 0), w.width)
		player.Position.Y = min(max(player.Position.Y, 0), w.height)
	}
}

func (w *World) Snapshot() Snapshot {
	players := make([]Player, 0, len(w.players))
	for _, player := range w.players {
		players = append(players, *player)
	}
	return Snapshot{Players: players}
}
