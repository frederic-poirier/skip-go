package main

import (
	"encoding/json"
	"errors"
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

	ready            chan *Player
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
	WinnerID      string
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
		query:      make(chan roomQuery, 16),
		register:   make(chan *Room, 16),
		unregister: make(chan *Room, 16),
	}
}

func (h *Hub) NewRoom() *Room {
	room := &Room{
		Hub:              h,
		ID:               generateRoomID(),
		Players:          make(map[string]*Player),
		ready:            make(chan *Player, 4),
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
		case player := <-r.ready:
			player.Ready = !player.Ready
			r.broadcastRoomView()

			// if !player.Ready is truthy it means player is not ready
			if len(r.Players) < 2 || len(r.Players) > 6 || !player.Ready || r.Game != nil {
				break
			}

			allReady := true
			for _, v := range r.Players {
				if !v.Ready {
					allReady = false
					break
				}
			}

			if !allReady {
				break
			}

			players := make([]Player, 0, len(r.Players))
			for _, p := range r.Players {
				players = append(players, *p)
			}

			game, err := NewGame(players, time.Now().UnixNano())
			if err != nil {
				r.Broadcast(errorMessage(err))
				break
			}

			r.Game = game
			game.Start()
			r.broadcastGameView()
		}
	}
}

func (r *Room) Broadcast(msg Message) {
	for i := range r.Players {
		r.sendTo(r.Players[i], msg)
	}
}

func (r *Room) NewClient(playerID string, conn *websocket.Conn) *Client {
	c := &Client{
		conn:     conn,
		send:     make(chan Message, 16),
		playerID: playerID,
		room:     r,
	}

	r.register <- c
	return c
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
		// do not register another player if already full.
		// do not register another player if already playing.
		if len(r.Players) == 6 || r.Game != nil {
			client.conn.Close()
			return
		}

		player = &Player{
			ID:     client.playerID,
			Score:  0,
			Client: client,
			Ready:  false,
		}
		r.Players[player.ID] = player
		r.broadcastRoomView()
		return
	}

	if player.Client != nil {
		player.Client.conn.Close()
	}

	player.Client = client
	r.sendRoomView(*player) // only the client change, hes the only one that need the current room view.
}

func (r *Room) UnregisterClient(client *Client) {
	for _, v := range r.Players {
		if v.Client == client {
			v.Client.conn.Close()
			v.Client = nil
			v.Ready = false
			r.broadcastRoomView()
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
		return
	}

	r.broadcastRoomView()
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

type RoomView struct {
	RoomID  string       `json:"roomId"`
	ID      string       `json:"id"`
	Ready   bool         `json:"isReady"`
	Players []PlayerView `json:"players"`
}

type PlayerView struct {
	ID    string `json:"id"`
	Score int    `json:"score"`
	Ready bool   `json:"isReady"`
}

func (r *Room) sendRoomView(player Player) {
	players := make([]PlayerView, 0, len(r.Players))
	for _, otherPlayer := range r.Players {
		if *otherPlayer == player {
			continue
		}

		players = append(players, PlayerView{
			ID:    otherPlayer.ID,
			Score: otherPlayer.Score,
			Ready: otherPlayer.Ready,
		})
	}

	payload, err := json.Marshal(RoomView{
		RoomID:  r.ID,
		ID:      player.ID,
		Ready:   player.Ready,
		Players: players,
	})
	if err != nil {
		log.Printf("error invalid json from room.view: %v", err)
	}

	r.sendTo(&player, Message{Type: "room-state", Payload: payload})
}

func (r *Room) broadcastRoomView() {
	for _, p := range r.Players {
		r.sendRoomView(*p)
	}
}

func (r *Room) endGame() {
	winner := r.Players[r.Game.WinnerID]
	if winner != nil {
		winner.Score += r.Game.score()
	}
	// TODO: broadcast to every player that winnerid won. and the score it got.
	r.Game = nil
	for _, p := range r.Players {
		p.Ready = false
	}
	r.broadcastRoomView()
}

func (r *Room) Handle(cmd Command) {
	player, ok := r.Players[cmd.PlayerID]
	if !ok || player.Client == nil {
		return
	}

	types := strings.Split(cmd.Message.Type, ".")
	if len(types) < 2 {
		return
	}

	prefix := types[0]
	suffix := types[1]
	switch prefix {
	case "game":
		if r.Game == nil {
			return
		}

		err := GameActionRouter(r.Game, player, suffix, cmd.Message.Payload)
		if err != nil {
			r.sendTo(player, errorMessage(err))
			return
		}

		if r.Game.IsOver() {
			r.endGame()
			return
		}

		r.broadcastGameView()

	case "lobby":
		if suffix == "ready" {
			r.ready <- player
		}
	default:
		err := errors.New("Could not find the action you requested")
		r.sendTo(player, errorMessage(err))
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
