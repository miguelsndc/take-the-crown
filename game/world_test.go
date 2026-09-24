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

	player, exists := world.Player("miguel")
	if !exists {
		t.Fatal("player should exist")
	}

	if player.Position != (Vec2{X: 500, Y: 300}) {
		t.Fatalf("unexpected spawn position: %+v", player.Position)
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

func TestPlayerMovement(t *testing.T) {
	world := NewWorld(1000, 600)
	world.AddPlayer("miguel")
	world.SetInput("miguel", Input{Right: true})

	world.Update(0.5)

	player, _ := world.Player("miguel")

	if player.Position != (Vec2{X: 600, Y: 300}) {
		t.Fatalf("unexpected position: %+v", player.Position)
	}
}

func TestDiagonalMovementIsNormalized(t *testing.T) {
	world := NewWorld(1000, 600)
	world.AddPlayer("miguel")
	world.SetInput("miguel", Input{
		Right: true,
		Down:  true,
	})

	before, _ := world.Player("miguel")
	world.Update(1)
	after, _ := world.Player("miguel")

	dx := after.Position.X - before.Position.X
	dy := after.Position.Y - before.Position.Y
	distance := math.Hypot(dx, dy)

	if math.Abs(distance-PLAYER_SPEED) > 0.000001 {
		t.Fatalf(
			"expected distance %v, got %v",
			PLAYER_SPEED,
			distance,
		)
	}
}

func TestPlayerCannotLeaveWorld(t *testing.T) {
	world := NewWorld(100, 100)
	world.AddPlayer("miguel")
	world.SetInput("miguel", Input{
		Up:   true,
		Left: true,
	})

	world.Update(10)

	player, _ := world.Player("miguel")

	if player.Position != (Vec2{X: 0, Y: 0}) {
		t.Fatalf("expected position (0, 0), got %+v", player.Position)
	}
}
