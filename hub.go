package main

import (
	"encoding/json"
	"math/rand"
)

type Hub struct {
	Rooms map[string]*Room
}

type Room struct {
	ID      string
	Players map[string]*Player
	Game    *Game
}

type Player struct {
	ID    string
	Score int
	Name  string
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

type Enveloppe struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}
