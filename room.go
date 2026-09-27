package main

import (
	"errors"
	"math/rand"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type Room struct {
	Hub     *Hub
	ID      RoomID
	Players map[PlayerID]*Player
	EmptyAt time.Time
	HostID  PlayerID
	Game    *Game

	Commands         chan Command
	registerPlayer   chan RegisterPlayerRequest
	registerClient   chan *Client
	unregisterClient chan *Client
	unregisterPlayer chan unregisterPlayerRequest
	checkEmpty       chan struct{}
}

type RegisterPlayerRequest struct {
	ID    PlayerID
	Reply chan error
}

type unregisterPlayerRequest struct {
	playerID PlayerID
	force    bool
}

type Command struct {
	PlayerID PlayerID
	Message  ClientMessage
}

type Player struct {
	ID             PlayerID
	CreatedAt      time.Time
	DisconnectedAt time.Time
	Score          int
	Client         *Client
}

type Client struct {
	conn     *websocket.Conn
	send     chan ServerMessage
	playerID PlayerID
	room     *Room
}

type Game struct {
	Players       []GamePlayer
	TurnIndex     int
	RNG           *rand.Rand
	Deck          Deck
	BuildPiles    BuildPiles
	CompletedPile Pile
	WinnerID      PlayerID
}

type GamePlayer struct {
	ID           PlayerID
	Hand         Hand
	StockPile    StockPile
	DiscardPiles DiscardPiles
}

func generateRoomID() RoomID {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 6)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return RoomID(b)
}

func (r *Room) run() {
	for {
		select {
		case command := <-r.Commands:
			r.Handle(command)
		case request := <-r.registerPlayer:
			r.RegisterPlayer(request)
		case client := <-r.registerClient:
			r.RegisterClient(client)
		case client := <-r.unregisterClient:
			r.UnregisterClient(client)
		case request := <-r.unregisterPlayer:
			r.UnregisterPlayer(request)
		case <-r.checkEmpty:
			r.checkEmptyRoom()
		}
	}
}

func (r *Room) checkEmptyRoom() {
	if r.EmptyAt.IsZero() {
		return
	}

	if time.Now().Before(r.EmptyAt.Add(30 * time.Second)) {
		return
	}

	r.Hub.RemoveRoom(r.ID)
}

func (r *Room) Broadcast(msg ServerMessage) {
	for i := range r.Players {
		r.sendTo(r.Players[i], msg)
	}
}

func (r *Room) RegisterPlayer(request RegisterPlayerRequest) {
	if _, ok := r.Players[request.ID]; ok {
		request.Reply <- nil
		return
	}

	if r.Game != nil {
		request.Reply <- errors.New("could not register the player in the room, the game is already started")
		return
	}

	if len(r.Players) == 6 {
		request.Reply <- errors.New("could not register the player in the room, the room is already full")
		return
	}

	r.Players[request.ID] = &Player{
		ID:        request.ID,
		CreatedAt: time.Now(),
		Score:     0,
		Client:    nil,
	}

	request.Reply <- nil
}

func (r *Room) RegisterClient(client *Client) {
	player := r.Players[client.playerID]
	if player.Client != nil {
		player.Client.conn.Close()
	}

	player.Client = client
	r.broadcastRoomView()

	if r.Game != nil {
		r.broadcastGameView()
	}
}

func (r *Room) sendTo(player *Player, msg ServerMessage) {
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

func (r *Room) UnregisterClient(client *Client) {
	for _, v := range r.Players {
		if v.Client == client {
			v.Client.conn.Close()
			v.Client = nil
			v.DisconnectedAt = time.Now()

			id := v.ID
			time.AfterFunc(30*time.Second, func() {
				r.unregisterPlayer <- unregisterPlayerRequest{
					playerID: id,
					force:    false,
				}
			})
		}
	}

	r.broadcastRoomView()
}

func (r *Room) UnregisterPlayer(request unregisterPlayerRequest) {
	id := request.playerID
	p, ok := r.Players[id]
	if !ok {
		return // joueur inconnu, rien est fait.
	}

	if !request.force {
		hasClient := p.Client != nil
		notTimedOut := time.Now().Before(p.DisconnectedAt.Add(30 * time.Second))
		if hasClient || notTimedOut {
			return
		}
	}

	delete(r.Players, id)
	r.Hub.RemovePlayer(id)

	if len(r.Players) == 0 {
		r.EmptyAt = time.Now()
		time.AfterFunc(30*time.Second, func() {
			if !time.Now().Before(r.EmptyAt.Add(30 * time.Second)) {
				r.checkEmpty <- struct{}{}
			}
		})
		// si la salle est vide donc rien a broadcast.
		return
	}

	if r.HostID != id {
		r.broadcastRoomView()
		return
	}

	var oldestPlayerID PlayerID
	var oldest time.Time
	for id, p := range r.Players {
		if oldestPlayerID == "" || p.CreatedAt.Before(oldest) {
			oldestPlayerID = id
			oldest = p.CreatedAt
		}
	}

	r.HostID = oldestPlayerID
	r.broadcastRoomView()
}

func (r *Room) broadcastGameView() {
	for _, p := range r.Players {
		r.sendGameView(p)
	}
}

func (r *Room) broadcastRoomView() {
	for id := range r.Players {
		r.sendRoomView(r.Players[id])
	}
}

func (r *Room) sendGameView(p *Player) {
	player, ok := r.Game.playerByID(p.ID)
	if !ok || p.Client == nil {
		return
	}

	r.sendTo(p, ServerMessage{
		Type:    ServerMessageTypeGameState,
		Payload: r.Game.view(*player),
	})
}

func (r *Room) sendRoomView(player *Player) {
	players := make([]PlayerView, 0, len(r.Players))
	for _, otherPlayer := range r.Players {
		if otherPlayer.ID == player.ID {
			continue
		}

		players = append(players, PlayerView{
			ID:        otherPlayer.ID,
			Score:     otherPlayer.Score,
			Connected: otherPlayer.Client != nil,
		})
	}

	r.sendTo(player, ServerMessage{
		Type: ServerMessageTypeRoomState,
		Payload: RoomView{
			RoomID:  r.ID,
			IsHost:  r.HostID == player.ID,
			ID:      player.ID,
			Players: players,
		},
	})
}

func (r *Room) startGame() {
	players := make([]Player, 0, len(r.Players))
	for _, p := range r.Players {
		players = append(players, *p)
	}

	game, err := NewGame(players, time.Now().UnixNano())
	if err != nil {
		host := r.Players[r.HostID]
		r.sendTo(host, errorMessage(err))
		return
	}

	r.Game = game
	game.Start()
	r.broadcastGameView()
}

func (r *Room) endGame() {
	winner := r.Players[r.Game.WinnerID]
	if winner != nil {
		winner.Score += r.Game.score()
	}
	// TODO: broadcast to every player that winnerid won. and the score it got.
	r.Game = nil
}

func (r *Room) Handle(cmd Command) {
	player, ok := r.Players[cmd.PlayerID]
	if !ok || player.Client == nil {
		return
	}

	msgType := string(cmd.Message.Type)
	types := strings.Split(msgType, ".")
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

		err := GameActionRouter(r.Game, player, cmd.Message.Type, cmd.Message.Payload)
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
		if suffix == "start" {
			if r.Game == nil {
				if cmd.PlayerID == r.HostID {
					r.startGame()
				} else {
					err := errors.New("only host can start game")
					r.sendTo(r.Players[cmd.PlayerID], errorMessage(err))
				}
			} else {
				err := errors.New("game is already running")
				r.sendTo(r.Players[cmd.PlayerID], errorMessage(err))
			}
		}
	default:
		err := errors.New("Could not find the action you requested")
		r.sendTo(player, errorMessage(err))
	}
}

func errorMessage(err error) ServerMessage {
	return ServerMessage{
		Type:    ServerMessageTypeError,
		Payload: err.Error(),
	}
}
