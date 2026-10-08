package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/miguelsndc/multiplayer-server/game"
	"github.com/miguelsndc/multiplayer-server/room"
)

type welcomeMessage struct {
	Type        string  `json:"type"`
	PlayerID    string  `json:"player_id"`
	WorldWidth  float64 `json:"world_width"`
	WorldHeight float64 `json:"world_height"`
}

type clientMessage struct {
	Type  string `json:"type"`
	Up    bool   `json:"up"`
	Down  bool   `json:"down"`
	Left  bool   `json:"left"`
	Right bool   `json:"right"`
	Slot  int    `json:"slot"`
}

var nextPlayerID atomic.Uint64

const (
	worldWidth  = 3840.0
	worldHeight = 2160.0
)

func handleWebSocket(gameRoom *room.Room) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			log.Printf("websocket upgrade failed: %v", err)
			return
		}

		defer conn.CloseNow()

		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		playerID := generateID()
		client := room.NewClient(playerID)

		if err := gameRoom.Join(ctx, client); err != nil {
			log.Printf("failed to join room: %v", err)
			return
		}

		defer func() {
			leaveCtx, leaveCancel := context.WithTimeout(
				context.Background(),
				time.Second,
			)
			defer leaveCancel()

			if err := gameRoom.Leave(leaveCtx, playerID); err != nil {
				log.Printf("failed to leave room: %v", err)
			}
		}()

		welcome := welcomeMessage{
			Type:        "welcome",
			PlayerID:    playerID,
			WorldWidth:  worldWidth,
			WorldHeight: worldHeight,
		}

		if err := wsjson.Write(ctx, conn, welcome); err != nil {
			log.Printf("failed to write welcome message: %v", err)
			return
		}

		go func() {
			if err := writeSnapshots(ctx, conn, client); err != nil {
				log.Printf(
					"snapshot writer for %s stopped: %v",
					playerID,
					err,
				)
			}
			cancel()
		}()

		for {
			var message clientMessage
			if err := wsjson.Read(ctx, conn, &message); err != nil {
				log.Printf(
					"websocket reader for %s stopped: %v",
					playerID,
					err,
				)
				return
			}

			switch message.Type {
			case "input":
				input := game.Input{
					Up:    message.Up,
					Down:  message.Down,
					Left:  message.Left,
					Right: message.Right,
				}
				if err := gameRoom.SetInput(ctx, playerID, input); err != nil {
					log.Printf("failed to set input for %s: %v", playerID, err)
					return
				}
			case "use_item":
				if err := gameRoom.UseItem(ctx, playerID, message.Slot); err != nil {
					log.Printf("failed to use item for %s: %v", playerID, err)
					return
				}
			}
		}
	}
}
func writeSnapshots(ctx context.Context, conn *websocket.Conn, client *room.Client) error {
	for {
		select {
		case snapshot, open := <-client.Snapshots():
			if !open {
				return nil
			}
			if err := wsjson.Write(ctx, conn, snapshot); err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func generateID() string {
	id := nextPlayerID.Add(1)
	return fmt.Sprintf("player-%d", id)
}

func Run() error {
	roomCtx, cancelRoom := context.WithCancel(context.Background())
	defer cancelRoom()
	gameRoom := room.NewRoom(worldWidth, worldHeight, 30)
	go gameRoom.Run(roomCtx)
	const (
		port = 3000
	)
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", handleWebSocket(gameRoom))
	fileServer := http.FileServer(http.Dir("./web"))
	mux.Handle("/", fileServer)
	actualPort := ":" + strconv.Itoa(port)
	listener, err := net.Listen("tcp", actualPort)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Server listening at port %d", port)
	return http.Serve(listener, mux)
}
