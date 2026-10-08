package room

import (
	"context"
	"time"

	"github.com/miguelsndc/multiplayer-server/game"
)

type eventType uint8

const (
	joinEvent eventType = iota
	leaveEvent
	inputEvent
	useItemEvent
)

type PlayerSnapshot struct {
	ID          string    `json:"id"`
	X           float64   `json:"x"`
	Y           float64   `json:"y"`
	CrownTime   float64   `json:"crown_time"`
	Inventory   [2]string `json:"inventory"`
	SpeedActive bool      `json:"speed_active"`
	GhostActive bool      `json:"ghost_active"`
}

type Snapshot struct {
	Type             string             `json:"type"`
	Tick             uint64             `json:"tick"`
	Players          []PlayerSnapshot   `json:"players"`
	Crown            game.CrownSnapshot `json:"crown"`
	Pickups          []game.Pickup       `json:"pickups"`
	Status           string              `json:"status"`
	WinnerID         string              `json:"winner_id"`
	RestartRemaining float64             `json:"restart_remaining"`
	TargetTime       float64             `json:"target_time"`
}

type Client struct {
	ID        string
	snapshots chan Snapshot
}

func NewClient(id string) *Client {
	return &Client{
		ID:        id,
		snapshots: make(chan Snapshot, 1),
	}
}

func (c *Client) Snapshots() <-chan Snapshot {
	return c.snapshots
}

func (c *Client) offer(snapshot Snapshot) {
	select {
	case c.snapshots <- snapshot:
		return
	default:
	}
	select {
	case <-c.snapshots:
	default:
	}
	select {
	case c.snapshots <- snapshot:
	default:
	}
}

type event struct {
	kind     eventType
	playerID string
	client   *Client
	input    game.Input
	slot     int
}

type Room struct {
	world        *game.World
	events       chan event
	tick         uint64
	tickInterval time.Duration
	clients      map[string]*Client
}

func NewRoom(width, height float64, tickRate int) *Room {
	if tickRate <= 0 {
		panic("tick rate has to be positive")
	}
	world := game.NewWorld(width, height)
	tickInterval := time.Second / time.Duration(tickRate)
	return &Room{
		world:        world,
		events:       make(chan event, 128),
		tickInterval: tickInterval,
		clients:      make(map[string]*Client),
	}
}

func (r *Room) broadcast() {
	worldSnapshot := r.world.Snapshot()
	players := make([]PlayerSnapshot, 0, len(worldSnapshot.Players))
	for _, player := range worldSnapshot.Players {
		players = append(players, PlayerSnapshot{
			ID:          player.ID,
			X:           player.Position.X,
			Y:           player.Position.Y,
			CrownTime:   player.CrownTime,
			Inventory:   player.Inventory,
			SpeedActive: player.SpeedRemaining > 0,
			GhostActive: player.GhostRemaining > 0,
		})
	}
	snapshot := Snapshot{
		Type:             "snapshot",
		Tick:             r.tick,
		Players:          players,
		Crown:            worldSnapshot.Crown,
		Pickups:          worldSnapshot.Pickups,
		Status:           worldSnapshot.Status,
		WinnerID:         worldSnapshot.WinnerID,
		RestartRemaining: worldSnapshot.RestartRemaining,
		TargetTime:       worldSnapshot.TargetTime,
	}
	for _, client := range r.clients {
		client.offer(snapshot)
	}
}

func (r *Room) Join(ctx context.Context, client *Client) error {
	return r.send(ctx, event{
		client: client,
		kind:   joinEvent,
	})
}

func (r *Room) closeAllClients() {
	for _, client := range r.clients {
		delete(r.clients, client.ID)
		r.world.RemovePlayer(client.ID)
		close(client.snapshots)
	}
}

func (r *Room) Leave(ctx context.Context, playerID string) error {
	return r.send(ctx, event{
		kind:     leaveEvent,
		playerID: playerID,
	})
}

func (r *Room) SetInput(
	ctx context.Context,
	playerID string,
	input game.Input,
) error {
	return r.send(ctx, event{
		kind:     inputEvent,
		playerID: playerID,
		input:    input,
	})
}

func (r *Room) UseItem(ctx context.Context, playerID string, slot int) error {
	return r.send(ctx, event{
		kind:     useItemEvent,
		playerID: playerID,
		slot:     slot,
	})
}

func (r *Room) send(ctx context.Context, e event) error {
	select {
	case r.events <- e:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *Room) removeClient(playerID string) {
	client, exists := r.clients[playerID]
	if !exists {
		return
	}
	delete(r.clients, playerID)
	r.world.RemovePlayer(playerID)
	close(client.snapshots)
}

func (r *Room) handle(e event) {
	switch e.kind {
	case joinEvent:
		client := e.client
		if !r.world.AddPlayer(client.ID) {
			close(client.snapshots)
			return
		}
		r.clients[client.ID] = client
	case leaveEvent:
		r.removeClient(e.playerID)
	case inputEvent:
		r.world.SetInput(e.playerID, e.input)
	case useItemEvent:
		r.world.UseItem(e.playerID, e.slot)
	}
}

func (r *Room) Run(ctx context.Context) {
	ticker := time.NewTicker(r.tickInterval)
	defer ticker.Stop()
	defer r.closeAllClients()
	for {
		select {
		case event := <-r.events:
			r.handle(event)
		case <-ticker.C:
			r.world.Update(r.tickInterval.Seconds())
			r.tick++
			r.broadcast()
		case <-ctx.Done():
			return
		}
	}
}
