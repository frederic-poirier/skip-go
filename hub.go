package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type Hub struct {
	Rooms      map[string]*Room
	register   chan *Room
	unregister chan *Room
	query      chan roomQuery
}

type roomQuery struct {
	ID    string
	Reply chan *Room
}

type Room struct {
	Hub     *Hub
	ID      string
	Players map[string]*Player
	Game    *Game

	Commands         chan Command
	register         chan *Client
	unregisterClient chan *Client
	unregisterPlayer chan *Player
}

type Command struct {
	PlayerID string
	Message  Message
}

type Player struct {
	ID     string
	Score  int
	Ready  bool
	Client *Client
}

type Client struct {
	conn     *websocket.Conn
	send     chan Message
	playerID string
	room     *Room
}

type Game struct {
	Players       []GamePlayer
	TurnIndex     int
	RNG           *rand.Rand
	Deck          Deck
	BuildPiles    BuildPiles
	CompletedPile Pile
}

type GamePlayer struct {
	ID           string
	Hand         Hand
	StockPile    StockPile
	DiscardPiles DiscardPiles
}

type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// server start
// 	-> create hub
//  -> hub.run()
//
// /room/create
//  -> create joinRequest
//  -> create room
//  -> room.register(joinRequest)
//  -> hub.register(room)
//  -> room.run()
//
// /room/:id/join
// 	-> find room
// 	-> create joinRequest
// 	-> room.register(joinRequest)

func (h *Hub) run() {
	for {
		select {
		case room := <-h.register:
			h.Rooms[room.ID] = room
		case room := <-h.unregister:
			delete(h.Rooms, room.ID)
		case query := <-h.query:
			room := h.Rooms[query.ID]
			query.Reply <- room
		}
	}
}

func (h *Hub) getRoom(id string) *Room {
	reply := make(chan *Room)
	h.query <- roomQuery{
		ID:    id,
		Reply: reply,
	}
	room := <-reply
	return room
}

func NewHub() *Hub {
	return &Hub{
		Rooms:      make(map[string]*Room),
		register:   make(chan *Room, 16),
		unregister: make(chan *Room, 16),
	}
}

func (h *Hub) NewRoom() *Room {
	room := &Room{
		Hub:              h,
		ID:               generateRoomID(),
		Players:          make(map[string]*Player),
		Commands:         make(chan Command, 16),
		register:         make(chan *Client, 4),
		unregisterClient: make(chan *Client, 4),
		unregisterPlayer: make(chan *Player, 4),
	}

	h.register <- room
	return room
}

// [todo] verifier que le roomID n'est pas utiliser deja.
func generateRoomID() string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 6)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

func (r *Room) run() {
	for {
		select {
		case command := <-r.Commands:
			r.Handle(command)
		case client := <-r.register:
			r.Register(client)
		case client := <-r.unregisterClient:
			r.UnregisterClient(client)
		case player := <-r.unregisterPlayer:
			r.UnregisterPlayer(player)
		}
	}
}

func (r *Room) sendTo(player *Player, msg Message) {
	if player.Client == nil {
		return
	}

	select {
	case player.Client.send <- msg:
	default:
		player.Client.conn.Close()
		player.Client = nil
	}
}

func (r *Room) Register(client *Client) {
	player, ok := r.Players[client.playerID]
	if !ok {
		player = &Player{
			ID:     client.playerID,
			Score:  0,
			Client: client,
			Ready:  false,
		}
		r.Players[player.ID] = player
		return
	}

	if player.Client != nil {
		player.Client.conn.Close()
	}
	player.Client = client
}

func (r *Room) UnregisterClient(client *Client) {
	for _, v := range r.Players {
		if v.Client == client {
			v.Client.conn.Close()
			v.Client = nil
		}
	}
}

func (r *Room) UnregisterPlayer(player *Player) {
	for _, v := range r.Players {
		if v == player {
			delete(r.Players, v.ID)
		}
	}

	if len(r.Players) == 0 {
		r.Hub.unregister <- r
	}
}

func (r *Room) broadcastGameView() {
	for _, p := range r.Game.Players {
		player, ok := r.Players[p.ID]
		if !ok || player.Client == nil {
			continue
		}

		payload, err := json.Marshal(r.Game.view(p))
		if err != nil {
			log.Printf("error invalid json from game.view: %v", err)
			continue
		}

		r.sendTo(player, Message{Type: "game-state", Payload: payload})

	}
}

func (r *Room) Handle(cmd Command) {
	player, ok := r.Players[cmd.PlayerID]
	if !ok || player.Client == nil {
		return
	}

	prefix := strings.Split(cmd.Message.Type, ".")[0]
	switch prefix {
	case "game":
		err := GameActionRouter(r.Game, *player, cmd.Message)
		if err != nil {
			r.sendTo(player, errorMessage(err))
		} else {
			r.broadcastGameView()
		}
	}
}

func errorMessage(err error) Message {
	payload, err := json.Marshal(err.Error())
	if err != nil {
		log.Printf("erreur malformaté: %v", err)
		return Message{}
	}

	return Message{
		Type:    "error",
		Payload: payload,
	}
}

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
)

func (c *Client) readPump() {
	defer func() {
		c.room.unregisterClient <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		var msg Message
		if err := c.conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Printf("erreur inattendue: %v", err)
			}
			break
		}
		c.room.Commands <- Command{PlayerID: c.playerID, Message: msg}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteJSON(msg); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
