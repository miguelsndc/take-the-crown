# Take the Crown

A small real-time multiplayer game built to study authoritative servers in Go.
The browser only sends player input. A room running at 30 ticks per second owns
the world state, updates the simulation and broadcasts snapshots to every
connected client over WebSockets.

## The game

Open the game in two browser tabs. The round starts when the second player
joins.

- Move with `WASD`.
- Touch the crown to pick it up.
- Touch the current holder to steal it.
- Hold the crown for 20 seconds to win the round.
- Collect up to two items and use them with `Q` and `E`.

The cyan item gives a temporary speed boost. The purple item makes the player
intangible for a short period, preventing other players from stealing the crown
from them.

## Running locally

```bash
go run .
```

Then open [http://localhost:3000](http://localhost:3000) in two tabs or browser
windows.

Run the game tests with:

```bash
go test ./...
go test -race ./...
```

## Structure

- `game/` contains the deterministic world simulation, crown rules and items.
- `room/` owns the world and serializes joins, leaves and input events.
- `server/` handles HTTP and WebSocket connections.
- `web/` contains the canvas client and input handling.

Slow clients do not block the room. Each client keeps a single pending snapshot;
when it falls behind, the old snapshot is replaced by the newest one.
