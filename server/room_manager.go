package server

import (
	"context"
	"sync"
	"time"

	"github.com/miguelsndc/multiplayer-server/game"
	"github.com/miguelsndc/multiplayer-server/room"
)

type managedRoom struct {
	gameRoom *room.Room
	players  int
	cancel   context.CancelFunc
}

type roomManager struct {
	mu    sync.Mutex
	ctx   context.Context
	rooms []*managedRoom
}

func newRoomManager(ctx context.Context) *roomManager {
	return &roomManager{
		ctx: ctx,
	}
}

func (m *roomManager) join(
	ctx context.Context,
	client *room.Client,
) (*room.Room, func() error, error) {
	m.mu.Lock()

	sala := m.findRoom()
	if sala == nil {
		sala = m.createRoom()
	}

	sala.players++
	m.mu.Unlock()

	if err := sala.gameRoom.Join(ctx, client); err != nil {
		m.releaseSlot(sala)
		return nil, nil, err
	}

	var once sync.Once

	leave := func() error {
		var leaveErr error

		once.Do(func() {
			leaveCtx, cancel := context.WithTimeout(
				context.Background(),
				time.Second,
			)
			defer cancel()

			leaveErr = sala.gameRoom.Leave(leaveCtx, client.ID)
			m.releaseSlot(sala)
		})

		return leaveErr
	}

	return sala.gameRoom, leave, nil
}

func (m *roomManager) findRoom() *managedRoom {
	for _, sala := range m.rooms {
		if sala.players > 0 && sala.players < game.MinPlayers {
			return sala
		}
	}

	var escolhida *managedRoom

	for _, sala := range m.rooms {
		if sala.players >= game.MaxPlayers {
			continue
		}

		if escolhida == nil || sala.players > escolhida.players {
			escolhida = sala
		}
	}

	return escolhida
}

func (m *roomManager) createRoom() *managedRoom {
	roomCtx, cancel := context.WithCancel(m.ctx)
	gameRoom := room.NewRoom(worldWidth, worldHeight, tickRate)

	sala := &managedRoom{
		gameRoom: gameRoom,
		cancel:   cancel,
	}

	m.rooms = append(m.rooms, sala)
	go gameRoom.Run(roomCtx)

	return sala
}

func (m *roomManager) releaseSlot(sala *managedRoom) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if sala.players > 0 {
		sala.players--
	}

	if sala.players != 0 {
		return
	}

	for index, current := range m.rooms {
		if current != sala {
			continue
		}

		m.rooms = append(m.rooms[:index], m.rooms[index+1:]...)
		sala.cancel()
		return
	}
}
