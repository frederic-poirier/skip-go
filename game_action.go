package main

import (
	"errors"
	"math/rand"
	"slices"
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
	ErrIllegalMove     = errors.New("move is not legal")
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

func (g *Game) draw(player *GamePlayer) {
	handCardCount := len(player.Hand)
	if handCardCount == MaxHandCard {
		return
	}

	for range MaxHandCard - handCardCount {
		card, ok := g.tryDrawCard()
		if !ok {
			return
		}

		player.Hand = append(player.Hand, card)
	}
}

func (g *Game) tryDrawCard() (Card, bool) {
	card, err := g.Deck.Pop()
	if err == ErrDeckIsEmpty {
		if len(g.CompletedPile) == 0 {
			return card, false
		}

		g.Deck = Deck(g.CompletedPile)
		g.Deck.Shuffle(g.RNG)
		g.CompletedPile = Pile{}
		return g.tryDrawCard()
	}

	return card, true
}

func (g *Game) Discard(player *GamePlayer, cardIdx, pileIdx int) error {
	card, err := player.Hand.at(cardIdx)
	if err != nil {
		return err
	}

	pile, err := player.DiscardPiles.pile(pileIdx)
	if err != nil {
		return err
	}

	player.Hand = slices.Delete(player.Hand, cardIdx, cardIdx+1)
	player.DiscardPiles[pileIdx] = append(Pile{card}, pile...)
	return nil
}

func (g *Game) startNextTurn() {
	g.TurnIndex = (g.TurnIndex + 1) % len(g.Players)
	g.draw(&g.Players[g.TurnIndex])
}

func (g *Game) PlayFromHand(player *GamePlayer, cardIdx, pileIdx int) error {
	card, err := player.Hand.at(cardIdx)
	if err != nil {
		return err
	}

	pile, err := g.BuildPiles.pile(pileIdx)
	if err != nil {
		return err
	}

	if !CanPlay(card, pile) {
		return ErrIllegalMove
	}

	player.Hand = slices.Delete(player.Hand, cardIdx, cardIdx+1)
	g.BuildPiles[pileIdx] = append(pile, card)
	return nil
}
