package game

import "math/rand"

const (
	ItemSpeed = "speed"
	ItemGhost = "ghost"

	PickupRadius       = 12.0
	MAX_PICKUPS        = 6
	PICKUP_SPAWN_DELAY = 2.5
	SPEED_DURATION     = 3.0
	GHOST_DURATION     = 1.25
	SPEED_MULTIPLIER   = 1.55
)

type Pickup struct {
	ID       uint64 `json:"id"`
	Type     string `json:"type"`
	Position Vec2   `json:"-"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
}

func (w *World) UseItem(playerID string, slot int) bool {
	player, exists := w.players[playerID]
	if !exists || w.status != RoundPlaying {
		return false
	}
	if slot < 0 || slot >= len(player.Inventory) {
		return false
	}

	item := player.Inventory[slot]
	switch item {
	case ItemSpeed:
		player.SpeedRemaining = SPEED_DURATION
	case ItemGhost:
		player.GhostRemaining = GHOST_DURATION
	default:
		return false
	}

	player.Inventory[slot] = ""
	return true
}

func (w *World) resetPickups() {
	w.pickups = nil
	w.pickupSpawnRemaining = PICKUP_SPAWN_DELAY
	for len(w.pickups) < 4 {
		if !w.spawnPickup() {
			break
		}
	}
}

func (w *World) updatePickups(dt float64) {
	w.pickupSpawnRemaining -= dt
	if w.pickupSpawnRemaining <= 0 && len(w.pickups) < MAX_PICKUPS {
		w.spawnPickup()
		w.pickupSpawnRemaining = PICKUP_SPAWN_DELAY
	}

	for index := 0; index < len(w.pickups); {
		pickup := w.pickups[index]
		coletado := false

		for _, jogador := range w.orderedPlayers() {
			slot := firstEmptySlot(jogador.Inventory)
			if slot == -1 {
				continue
			}
			if circlesOverlap(jogador.Position, PlayerRadius, pickup.Position, PickupRadius) {
				jogador.Inventory[slot] = pickup.Type
				w.pickups = append(w.pickups[:index], w.pickups[index+1:]...)
				coletado = true
				break
			}
		}

		if !coletado {
			index++
		}
	}
}

func (w *World) spawnPickup() bool {
	const margin = 80.0
	if w.width <= margin*2 || w.height <= margin*2 {
		return false
	}

	for tentativa := 0; tentativa < 30; tentativa++ {
		center := w.crown.Position
		players := w.orderedPlayers()
		if len(players) > 0 {
			center = players[rand.Intn(len(players))].Position
		}
		position := Vec2{
			X: center.X + (rand.Float64()-0.5)*900,
			Y: center.Y + (rand.Float64()-0.5)*500,
		}
		position.X = min(max(position.X, margin), w.width-margin)
		position.Y = min(max(position.Y, margin), w.height-margin)
		if !w.pickupPositionAvailable(position) {
			continue
		}

		itemType := ItemSpeed
		if rand.Intn(2) == 1 {
			itemType = ItemGhost
		}

		w.nextPickupID++
		w.pickups = append(w.pickups, Pickup{
			ID:       w.nextPickupID,
			Type:     itemType,
			Position: position,
			X:        position.X,
			Y:        position.Y,
		})
		return true
	}
	return false
}

func (w *World) pickupPositionAvailable(position Vec2) bool {
	if circlesOverlap(position, PickupRadius+50, w.crown.Position, CrownRadius) {
		return false
	}
	for _, jogador := range w.players {
		if circlesOverlap(position, PickupRadius+30, jogador.Position, PlayerRadius) {
			return false
		}
	}
	for _, pickup := range w.pickups {
		if circlesOverlap(position, PickupRadius+30, pickup.Position, PickupRadius) {
			return false
		}
	}
	return true
}

func firstEmptySlot(inventory [2]string) int {
	for index, item := range inventory {
		if item == "" {
			return index
		}
	}
	return -1
}
