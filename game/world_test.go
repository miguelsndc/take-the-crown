package game

import (
	"math"
	"testing"
)

func TestPlayerLifecycle(t *testing.T) {
	world := NewWorld(1000, 600)

	if !world.AddPlayer("miguel") {
		t.Fatal("first insertion should succeed")
	}
	if world.status != RoundWaiting {
		t.Fatal("one player should not start the round")
	}
	if world.AddPlayer("miguel") {
		t.Fatal("duplicate insertion should fail")
	}
	if !world.RemovePlayer("miguel") {
		t.Fatal("removal should succeed")
	}
	if _, exists := world.Player("miguel"); exists {
		t.Fatal("removed player should not exist")
	}
}

func TestRoundStartsWithTwoPlayers(t *testing.T) {
	world := NewWorld(1000, 600)
	world.AddPlayer("player-1")
	world.AddPlayer("player-2")

	if world.status != RoundPlaying {
		t.Fatalf("expected playing state, got %s", world.status)
	}
	first, _ := world.Player("player-1")
	second, _ := world.Player("player-2")
	if first.Position == second.Position {
		t.Fatal("players should have different spawn positions")
	}
}

func TestPlayerMovement(t *testing.T) {
	world := NewWorld(1000, 600)
	world.AddPlayer("miguel")
	before, _ := world.Player("miguel")
	world.SetInput("miguel", Input{Right: true})

	world.Update(0.5)

	after, _ := world.Player("miguel")
	if math.Abs((after.Position.X-before.Position.X)-100) > 0.000001 {
		t.Fatalf("unexpected position: %+v", after.Position)
	}
}

func TestDiagonalMovementIsNormalized(t *testing.T) {
	world := NewWorld(1000, 600)
	world.AddPlayer("miguel")
	world.SetInput("miguel", Input{Right: true, Down: true})

	before, _ := world.Player("miguel")
	world.Update(1)
	after, _ := world.Player("miguel")

	dx := after.Position.X - before.Position.X
	dy := after.Position.Y - before.Position.Y
	distance := math.Hypot(dx, dy)
	if math.Abs(distance-PLAYER_SPEED) > 0.000001 {
		t.Fatalf("expected distance %v, got %v", PLAYER_SPEED, distance)
	}
}

func TestPlayerCannotLeaveWorld(t *testing.T) {
	world := NewWorld(100, 100)
	world.AddPlayer("miguel")
	world.SetInput("miguel", Input{Up: true, Left: true})

	world.Update(10)

	player, _ := world.Player("miguel")
	expected := Vec2{X: PlayerRadius, Y: PlayerRadius}
	if player.Position != expected {
		t.Fatalf("expected position %+v, got %+v", expected, player.Position)
	}
}

func TestCrownCanBePickedUpAndStolen(t *testing.T) {
	world := NewWorld(1000, 600)
	world.AddPlayer("player-1")
	world.AddPlayer("player-2")

	world.players["player-1"].Position = world.crown.Position
	world.Update(0.01)
	if world.crown.HolderID != "player-1" {
		t.Fatalf("expected player-1 to hold crown, got %s", world.crown.HolderID)
	}

	world.crown.CooldownRemaining = 0
	world.players["player-2"].Position = world.players["player-1"].Position
	world.Update(0.01)
	if world.crown.HolderID != "player-2" {
		t.Fatalf("expected player-2 to steal crown, got %s", world.crown.HolderID)
	}
}

func TestCrownCooldownAndGhostBlockStealing(t *testing.T) {
	world := NewWorld(1000, 600)
	world.AddPlayer("player-1")
	world.AddPlayer("player-2")

	first := world.players["player-1"]
	second := world.players["player-2"]
	first.Position = world.crown.Position
	second.Position = first.Position
	world.crown.HolderID = first.ID
	world.crown.CooldownRemaining = 1

	world.Update(0.1)
	if world.crown.HolderID != first.ID {
		t.Fatal("crown should not be stolen during cooldown")
	}

	world.crown.CooldownRemaining = 0
	first.GhostRemaining = 1
	world.Update(0.1)
	if world.crown.HolderID != first.ID {
		t.Fatal("crown should not be stolen from an intangible player")
	}
}

func TestWinningRestartsTheRound(t *testing.T) {
	world := NewWorld(1000, 600)
	world.AddPlayer("player-1")
	world.AddPlayer("player-2")

	winner := world.players["player-1"]
	world.crown.HolderID = winner.ID
	winner.CrownTime = CROWN_TARGET_TIME - 0.01
	world.Update(0.02)

	if world.status != RoundFinished || world.winnerID != winner.ID {
		t.Fatal("round should finish when a player reaches target time")
	}

	world.Update(ROUND_RESTART_DELAY + 0.1)
	if world.status != RoundPlaying {
		t.Fatal("round should restart after the delay")
	}
	if world.players["player-1"].CrownTime != 0 {
		t.Fatal("scores should reset between rounds")
	}
}

func TestUsingItemConsumesItsSlot(t *testing.T) {
	world := NewWorld(1000, 600)
	world.AddPlayer("player-1")
	world.AddPlayer("player-2")

	player := world.players["player-1"]
	player.Inventory[0] = ItemSpeed
	if !world.UseItem(player.ID, 0) {
		t.Fatal("player should be able to use an owned item")
	}
	if player.Inventory[0] != "" || player.SpeedRemaining != SPEED_DURATION {
		t.Fatal("speed item should be consumed and activate its effect")
	}
}
