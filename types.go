package main

import "encoding/json"

type (
	RoomID   string
	PlayerID string

	Card         int
	Pile         []Card
	Hand         []Card
	DiscardPiles [4]Pile
	BuildPiles   [4]Pile

	ServerMessageType string
	ClientMessageType string
)

const (
	ServerMessageTypeError     ServerMessageType = "error"
	ServerMessageTypeRoomState ServerMessageType = "room.state"
	ServerMessageTypeGameState ServerMessageType = "game.state"
)

type ServerMessage struct {
	Type    ServerMessageType `json:"type"`
	Payload any               `json:"payload"`
}

const (
	ClientMessageTypeLobbyStart          ClientMessageType = "lobby.start"
	ClientMessageTypeGamePlayFromHand    ClientMessageType = "game.playFromHand"
	ClientMessageTypeGamePlayFromStock   ClientMessageType = "game.playFromStock"
	ClientMessageTypeGamePlayFromDiscard ClientMessageType = "game.playFromDiscard"
	ClientMessageTypeGameDiscard         ClientMessageType = "game.discard"
)

type ClientMessage struct {
	Type    ClientMessageType `json:"type"`
	Payload json.RawMessage   `json:"payload"`
}

type GameView struct {
	IsPlayerTurn   bool         `json:"isPlayerTurn"`
	BuildPiles     BuildPiles   `json:"buildPiles"`
	DiscardPiles   DiscardPiles `json:"discardPiles"`
	Hand           Hand         `json:"hand"`
	StockPileCount int          `json:"stockPileCount"`
	StockPileCard  Card         `json:"stockPileTopCard"`
	Opponents      []Opponent   `json:"opponnents"`
}

type RoomView struct {
	RoomID  RoomID       `json:"roomId"`
	IsHost  bool         `json:"isHost"`
	ID      PlayerID     `json:"id"`
	Players []PlayerView `json:"players"`
}

type PlayerView struct {
	ID        PlayerID `json:"id"`
	Score     int      `json:"score"`
	Connected bool     `json:"connected"`
}

type Opponent struct {
	PlayerID       PlayerID     `json:"playerId"`
	HandCount      int          `json:"handCount"`
	StockPileCount int          `json:"stockPileCount"`
	StockPileCard  Card         `json:"stockPileCard"`
	DiscardPiles   DiscardPiles `json:"discardPiles"`
}

type PlayFromDiscardPayload struct {
	DiscardPileIdx int `json:"discardPileIdx"`
	BuildPileIdx   int `json:"buildPileIdx"`
}

type PlayFromStockPayload struct {
	BuildPileIdx int `json:"buildPileIdx"`
}

type PlayFromHandPayload struct {
	CardIdx      int `json:"cardIdx"`
	BuildPileIdx int `json:"buildPileIdx"`
}

type DiscardPayload struct {
	CardIdx        int `json:"cardIdx"`
	DiscardPileIdx int `json:"discardPileIdx"`
}

type ErrorPayload string
