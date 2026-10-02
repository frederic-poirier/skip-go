package main

import (
	"errors"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"
)

var (
	ERR_REGISTER_ROOM_FULL    = errors.New("room cannot register the player, the room is currently full")
	ERR_REGISTER_ROOM_IN_GAME = errors.New("room cannot register the player, the room is currently in game")
)

const (
	DISCONNECT_TIMEOUT = time.Second * 30
	EMPTY_ROOM_TIMEOUT = time.Second * 30
)

type Room struct {
	Hub     *Hub
	ID      RoomID
	Players map[PlayerID]*Player
	EmptyAt time.Time
	HostID  PlayerID
	Game    *Game

	Commands         chan Command
	register         chan *Client
	unregisterClient chan *Client
	checkPlayer      chan PlayerID
	checkEmpty       chan struct{}
	state            openState
}

type openState struct {
	isOpen bool
	mu     sync.Mutex
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

func NewRoom(hub *Hub, id RoomID) *Room {
	return &Room{
		Hub:              hub,
		ID:               id,
		Players:          make(map[PlayerID]*Player),
		EmptyAt:          time.Now(),
		Commands:         make(chan Command, 16),
		register:         make(chan *Client, 4),
		unregisterClient: make(chan *Client, 4),
		checkPlayer:      make(chan PlayerID, 4),
		checkEmpty:       make(chan struct{}, 4),
		state:            openState{isOpen: true},
	}
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
		case client := <-r.register:
			r.Register(client)
		case client := <-r.unregisterClient:
			r.UnregisterClient(client)
		case id := <-r.checkPlayer:
			r.checkDisconnectPlayer(id)
		case <-r.checkEmpty:
			r.checkEmptyRoom()
		}
	}
}

var (
	ERR_ROOM_COMMAND_BUFFER_FULL = errors.New("the room command buffer is currently full")
	ERR_ROOM_CLOSE               = errors.New("the room is close")
)

func (r *Room) send(cmd Command) error {
	r.state.mu.Lock()
	defer r.state.mu.Unlock()

	if !r.state.isOpen {
		return ERR_ROOM_CLOSE
	}

	select {
	case r.Commands <- cmd:
		return nil
	default:
		return ERR_ROOM_COMMAND_BUFFER_FULL
	}
}

func (r *Room) checkEmptyRoom() {
	r.state.mu.Lock()
	defer r.state.mu.Unlock()

	if !r.state.isOpen {
		return
	}

	if r.EmptyAt.IsZero() {
		return
	}

	if time.Now().Before(r.EmptyAt.Add(EMPTY_ROOM_TIMEOUT)) {
		return
	}

	r.state.isOpen = false
	r.Hub.RemoveRoom(r.ID)
}

func (r *Room) checkDisconnectPlayer(playerID PlayerID) {
	player, ok := r.Players[playerID]
	if !ok {
		return
	}

	if player.Client != nil || !time.Now().Before(player.DisconnectedAt.Add(DISCONNECT_TIMEOUT)) {
		return
	}

	r.UnregisterPlayer(playerID)
}

func (r *Room) Broadcast(msg ServerMessage) {
	for i := range r.Players {
		r.sendTo(r.Players[i], msg)
	}
}

func (r *Room) Register(client *Client) {
	if player, exist := r.Players[client.playerID]; exist {
		if player.Client != nil {
			r.UnregisterClient(player.Client)
		}

		player.Client = client
		player.DisconnectedAt = time.Time{}
		r.sendRoomView(player)
		r.sendGameView(player)
		return
	}

	if r.Game != nil {
		r.Hub.RemovePlayer(client.playerID)
		client.trySend(errorMessage(ERR_REGISTER_ROOM_IN_GAME))
		r.UnregisterClient(client)
		return
	}

	if len(r.Players) >= 6 {
		r.Hub.RemovePlayer(client.playerID)
		client.trySend(errorMessage(ERR_REGISTER_ROOM_FULL))
		r.UnregisterClient(client)
		return
	}

	player := &Player{
		ID:             client.playerID,
		CreatedAt:      time.Now(),
		DisconnectedAt: time.Time{},
		Score:          0,
		Client:         client,
	}

	r.Players[client.playerID] = player
	r.broadcastRoomView()
}

func (r *Room) sendTo(player *Player, msg ServerMessage) {
	if player.Client == nil {
		return
	}

	err := player.Client.trySend(msg)
	if err == ERR_CLIENT_BUFFER_FULL {
		r.UnregisterClient(player.Client)
	}
}

// shutdown the client, if this client is still
// attach to the player, remove it and set it as
// disconnected, send in 30 second a request to
// remove the player if still disconnected.
func (r *Room) UnregisterClient(client *Client) {
	client.shutdown()

	player, exist := r.Players[client.playerID]
	if !exist || player.Client != client {
		return
	}

	player.Client = nil
	player.DisconnectedAt = time.Now()

	time.AfterFunc(DISCONNECT_TIMEOUT, func() {
		r.checkPlayer <- client.playerID
	})

	r.broadcastRoomView()
}

func (r *Room) UnregisterPlayer(playerID PlayerID) {
	if _, ok := r.Players[playerID]; !ok {
		return // joueur inconnu, rien est fait.
	}

	delete(r.Players, playerID)
	r.Hub.RemovePlayer(playerID)

	if len(r.Players) == 0 {
		r.EmptyAt = time.Now()
		time.AfterFunc(EMPTY_ROOM_TIMEOUT, func() {
			r.checkEmpty <- struct{}{}
		})
		return
	}

	if r.HostID != playerID {
		r.broadcastRoomView()
		return
	}

	var oldestPlayerID PlayerID
	var oldest time.Time
	for id, p := range r.Players {
		log.Println(id)
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
	if r.Game == nil {
		return
	}

	player, ok := r.Game.playerByID(p.ID)
	if !ok {
		return
	}

	r.sendTo(p, ServerMessage{
		Type:    ServerMessageTypeGameState,
		Payload: r.Game.view(*player),
	})
}

func (r *Room) sendRoomView(player *Player) {
	players := make([]PlayerView, len(r.Players))
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
