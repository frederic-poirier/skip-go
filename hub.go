package main

import (
	"errors"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
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

func (h *Hub) NewRoom(playerID PlayerID) (*Room, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.roomIDByPlayer[playerID]; ok {
		return nil, errors.New("player is already inside a room")
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
		log.Println("After 5 try, the hub could not create a unique room ID")
		return nil, errors.New("Could not create a unique id for the room ID")
	}

	room := &Room{
		Hub:              h,
		ID:               id,
		Players:          make(map[PlayerID]*Player),
		EmptyAt:          time.Now(),
		Commands:         make(chan Command, 16),
		registerPlayer:   make(chan RegisterPlayerRequest, 4),
		registerClient:   make(chan *Client, 4),
		unregisterClient: make(chan *Client, 4),
		unregisterPlayer: make(chan unregisterPlayerRequest, 4),
		checkEmpty:       make(chan struct{}, 4),
	}

	log.Printf("new room [id: %v] created by [%v]\n", id, playerID)
	h.rooms[id] = room
	return room, nil
}

func (h *Hub) RemoveRoom(id RoomID) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.rooms, id)
}

func (h *Hub) Join(playerID PlayerID, roomID RoomID) (reconected bool, err error) {
	foundRoom, reconected, err := h.prepareJoin(playerID, roomID)
	if err != nil || reconected {
		return reconected, err
	}

	Reply := make(chan error)
	foundRoom.registerPlayer <- RegisterPlayerRequest{ID: playerID, Reply: Reply}
	if err = <-Reply; err != nil {
		h.RemovePlayer(playerID)
		return false, err
	}

	h.AddPlayer(playerID, roomID)
	log.Printf("new player [id: %v] joined room [id: %v]\n", playerID, roomID)
	return false, nil
}

func (h *Hub) prepareJoin(playerID PlayerID, roomID RoomID) (*Room, bool, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	foundRoomID, exist := h.roomIDByPlayer[playerID]
	if exist {
		if foundRoomID != roomID {
			err := errors.New("player is already registered in another roomID")
			return nil, false, err
		}

		_, exist := h.rooms[foundRoomID]
		if exist {
			return nil, true, nil
		}

		delete(h.roomIDByPlayer, playerID)
		err := errors.New("player is already registered in that roomID, but the room could not be found")
		return nil, false, err
	}

	foundRoom, exist := h.rooms[roomID]
	if !exist {
		err := errors.New("could not find the room with the roomID provided")
		return nil, false, err
	}

	return foundRoom, false, nil
}

func (h *Hub) Attach(playerID PlayerID, conn *websocket.Conn) (*Client, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	roomID, exist := h.roomIDByPlayer[playerID]
	if !exist {
		return nil, errors.New("could not find the roomID with the playerID provided")
	}

	room, exist := h.rooms[roomID]
	if !exist {
		delete(h.roomIDByPlayer, playerID)
		return nil, errors.New("could not find the room with the roomID associated to the player")
	}

	client := &Client{
		conn:     conn,
		send:     make(chan ServerMessage, 64),
		playerID: playerID,
		room:     room,
	}

	room.registerClient <- client
	log.Printf("new connection on player [id: %v] in the room [id: %v]\n", playerID, roomID)
	return client, nil
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
