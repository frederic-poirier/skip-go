package main

import "math/rand"

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
	StockPile    StockPiles
	DiscardPiles DiscardPiles
}
