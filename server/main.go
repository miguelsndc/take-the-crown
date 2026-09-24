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
	"github.com/miguelsndc/multiplayer-server/room"
)

type welcomeMessage struct {
	Type     string `json:"type"`
	PlayerID string `json:"player_id"`
}

var nextPlayerID atomic.Uint64

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
			Type:     "welcome",
			PlayerID: playerID,
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

				cancel()
			}
		}()

		for {
			_, message, err := conn.Read(ctx)
			if err != nil {
				log.Printf(
					"websocket reader for %s stopped: %v",
					playerID,
					err,
				)
				return
			}

			log.Printf(
				"received from %s: %s",
				playerID,
				message,
			)
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
	gameRoom := room.NewRoom(1000, 600, 30)
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
