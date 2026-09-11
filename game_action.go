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
	ErrPlayerNotFound  = errors.New("player could not be found")
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
			StockPile{},
			DiscardPiles{},
		})
	}

	return g, nil
}

func (g *Game) deal(stockPilesSize int) {
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
	card, err := g.Deck.Shift()
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

func (g *Game) PlayFromStock(player *GamePlayer, pileIdx int) error {
	if len(player.StockPile) == 0 {
		return ErrIllegalMove
	}

	card := player.StockPile[0]
	pile, err := g.BuildPiles.pile(pileIdx)
	if err != nil {
		return err
	}

	if !CanPlay(card, pile) {
		return ErrIllegalMove
	}

	player.StockPile = player.StockPile[1:]
	g.BuildPiles[pileIdx] = append(pile, card)
	return nil
}

func (g *Game) PlayFromDiscard(player *GamePlayer, dpIdx, bpIdx int) error {
	dp, err := player.DiscardPiles.pile(dpIdx)
	if err != nil {
		return ErrInvalidPileIndex
	}

	bp, err := g.BuildPiles.pile(bpIdx)
	if err != nil {
		return ErrInvalidPileIndex
	}

	card, err := dp.Top()
	if !CanPlay(card, bp) || err != nil {
		return ErrIllegalMove
	}

	player.DiscardPiles[dpIdx].Pop()
	g.BuildPiles[bpIdx] = append(bp, card)
	return nil
}

func (g *Game) findPlayerTurn() *GamePlayer {
	return &g.Players[g.TurnIndex]
}

func (g *Game) score() int {
	score := 25
	for _, p := range g.Players {
		// player that win should not have card
		// so the score would not add up.
		score += len(p.StockPile) * 5
	}
	return score
}

type GameView struct {
	TurnIndex      int          `json:"turnIndex"`
	BuildPiles     BuildPiles   `json:"buildPiles"`
	DiscardPiles   DiscardPiles `json:"discardPiles"`
	StockPileCount int          `json:"stockPileCount"`
	StockPileCard  Card         `json:"stockPileTopCard"`
	Opponents      []Opponent   `json:"opponnents"`
}

type Opponent struct {
	PlayerID       string       `json:"playerId"`
	HandCount      int          `json:"handCount"`
	StockPileCount int          `json:"stockPileCount"`
	StockPileCard  Card         `json:"stockPileCard"`
	DiscardPiles   DiscardPiles `json:"discardPiles"`
}

func (g *Game) view(p *GamePlayer) GameView {
	gv := GameView{
		TurnIndex:      g.TurnIndex,
		BuildPiles:     g.BuildPiles,
		DiscardPiles:   p.DiscardPiles,
		StockPileCount: len(p.StockPile),
		StockPileCard:  p.StockPile[0],
	}

	for _, o := range g.Players {
		gv.Opponents = append(gv.Opponents, Opponent{
			PlayerID:       o.ID,
			HandCount:      len(o.Hand),
			StockPileCount: len(o.StockPile),
			StockPileCard:  o.StockPile[0],
			DiscardPiles:   o.DiscardPiles,
		})
	}

	return gv
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

const (
	MsgTypePlayHand    = "playFromHand"
	MsgTypePlayStock   = "playFromStock"
	MsgTypePlayDiscard = "playFromDiscard"
	MsgTypeDiscard     = "discard"
)
