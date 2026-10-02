package main

import (
	"errors"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	ERR_PLAYER_IS_UNASSIGNED        = errors.New("the player is not assigned to any room")
	ERR_ALREADY_ASSIGNED_OTHER_ROOM = errors.New("the player is already assigned to another room")
	ERR_GENERATE_ID_FAIL            = errors.New("the roomd id generator could not create a unique id")
	ERR_ROOM_NOT_FOUND              = errors.New("the roomID provided does not match any room")
)

type Hub struct {
	rooms          map[RoomID]*Room
	roomIDByPlayer map[PlayerID]RoomID
	mu             sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		rooms:          make(map[RoomID]*Room),
		roomIDByPlayer: make(map[PlayerID]RoomID),
	}
}

func (h *Hub) CreateRoom(playerID PlayerID) (*Room, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exist := h.roomIDByPlayer[playerID]; exist {
		return nil, ERR_ALREADY_ASSIGNED_OTHER_ROOM
	}

	var id RoomID
	for range 5 {
		id = generateRoomID()
		if _, ok := h.rooms[id]; !ok {
			break
		}
		id = RoomID("")
	}

	if id == "" {
		return nil, ERR_GENERATE_ID_FAIL
	}

	room := NewRoom(h, id)
	h.rooms[id] = room
	return room, nil
}

func (h *Hub) RemoveRoom(id RoomID) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.rooms, id)
}

func (h *Hub) RemovePlayer(playerID PlayerID) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.roomIDByPlayer, playerID)
}

func (h *Hub) AddPlayer(playerID PlayerID, roomID RoomID) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.roomIDByPlayer[playerID] = roomID
}

func (h *Hub) AssignPlayer(playerID PlayerID, roomID RoomID) (reconected bool, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	_, exist := h.rooms[roomID]
	if !exist {
		return false, ERR_ROOM_NOT_FOUND
	}

	currentlyAssignRoomID, exist := h.roomIDByPlayer[playerID]
	if exist && currentlyAssignRoomID != roomID {
		return false, ERR_ALREADY_ASSIGNED_OTHER_ROOM
	}

	h.roomIDByPlayer[playerID] = roomID
	return exist, nil
}

func (h *Hub) Attach(playerID PlayerID, conn *websocket.Conn) (*Client, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	roomID, exist := h.roomIDByPlayer[playerID]
	if !exist {
		return nil, ERR_PLAYER_IS_UNASSIGNED
	}

	room, exist := h.rooms[roomID]
	if !exist {
		delete(h.roomIDByPlayer, playerID)
		return nil, ERR_ROOM_NOT_FOUND
	}

	client := NewClient(conn, playerID, room)
	room.register <- client
	return client, nil
}
