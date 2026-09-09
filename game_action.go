package main

import (
	"errors"
	"math/rand"
)

const (
	MaxPlayerNumber = 6
	MinPlayerNumber = 2
	MaxHandCard     = 5
)

var (
	ErrNotEnoughPlayer = errors.New("game need atleast 2 players")
	ErrTooMuchPlayer   = errors.New("game cannot have more than 6 players")
	ErrDrawHandFull    = errors.New("cannot draw, player's hand is already full")
)

func NewGame(players []Player, seed int64) (*Game, error) {
	if len(players) > MaxPlayerNumber {
		return nil, ErrTooMuchPlayer
	}

	if len(players) < MinPlayerNumber {
		return nil, ErrNotEnoughPlayer
	}

	g := &Game{
		Deck:          NewDeck(),
		TurnIndex:     0,
		RNG:           rand.New(rand.NewSource(seed)),
		BuildPiles:    BuildPiles{},
		CompletedPile: Pile{},
	}

	for _, player := range players {
		g.Players = append(g.Players, GamePlayer{
			player.ID,
			Hand{},
			StockPiles{},
			DiscardPiles{},
		})
	}

	return g, nil
}

func (g *Game) Deal(stockPilesSize int) {
	for range 5 {
		for i := range g.Players {
			g.Players[i].Hand = append(g.Players[i].Hand, g.Deck[0])
			g.Deck = g.Deck[1:]
		}
	}

	for range stockPilesSize {
		for i := range g.Players {
			g.Players[i].StockPile = append(g.Players[i].StockPile, g.Deck[0])
			g.Deck = g.Deck[1:]
		}
	}
}

func (g *Game) Draw(player *GamePlayer) error {
	handCardCount := len(player.Hand)
	if handCardCount == MaxHandCard {
		return ErrDrawHandFull
	}

	for range MaxHandCard - handCardCount {
		card, err := g.Deck.Pop()
		if err == ErrDeckIsEmpty {
			g.Deck = Deck(g.CompletedPile)
			g.Deck.Shuffle(g.RNG)
			card, err := g.Deck.Pop()
			if err != nil {
				return err
			}

			player.Hand = append(player.Hand, card)
		} else if err != nil {
			return err
		}

		player.Hand = append(player.Hand, card)
	}

	return nil
}
