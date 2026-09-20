package main

import (
	"encoding/json"
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
		WinnerID:      "",
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

func (g *Game) Start() {
	g.Deck.Shuffle(g.RNG)
	g.deal(10)
	g.startNextTurn()
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

func (g *Game) playOnToBuildPile(pile *Pile, card Card) error {
	*pile = append(*pile, card)
	if len(*pile) == 12 {
		g.CompletedPile = append(g.CompletedPile, *pile...)
		*pile = Pile{}
	}

	return nil
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
	g.playOnToBuildPile(&g.BuildPiles[pileIdx], card)
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
	g.playOnToBuildPile(&g.BuildPiles[pileIdx], card)

	if len(player.StockPile) == 0 {
		g.WinnerID = player.ID
	}

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
	g.playOnToBuildPile(&g.BuildPiles[bpIdx], card)
	return nil
}

func (g *Game) findPlayerTurn() *GamePlayer {
	return &g.Players[g.TurnIndex]
}

func (g *Game) IsOver() bool {
	return g.WinnerID != ""
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
	Hand           Hand         `json:"hand"`
	StockPileCount int          `json:"stockPileCount"`
	StockPileCard  Card         `json:"stockPileTopCard"`
	Opponents      []Opponent   `json:"opponnents"`
}

type Opponent struct {
	PlayerID       playerID     `json:"playerId"`
	HandCount      int          `json:"handCount"`
	StockPileCount int          `json:"stockPileCount"`
	StockPileCard  Card         `json:"stockPileCard"`
	DiscardPiles   DiscardPiles `json:"discardPiles"`
}

func (g *Game) view(p GamePlayer) GameView {
	gv := GameView{
		TurnIndex:      g.TurnIndex,
		BuildPiles:     g.BuildPiles,
		Hand:           p.Hand,
		DiscardPiles:   p.DiscardPiles,
		StockPileCount: len(p.StockPile),
		StockPileCard:  p.StockPile[0],
	}

	for _, o := range g.Players {
		if o.ID == p.ID {
			continue
		}

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
	ActionPlayFromHand    = "playFromHand"
	ActionPlayFromStock   = "playFromStock"
	ActionPlayFromDiscard = "playFromDiscard"
	ActionDiscard         = "discard"
)

var (
	ErrNotPlayerTurn   = errors.New("is not player's turn")
	ErrMsgTypeNotFound = errors.New("msg type could not be found")
)

func GameActionRouter(g *Game, player *Player, action string, payload json.RawMessage) error {
	gamePlayer := g.findPlayerTurn()
	if gamePlayer.ID != player.ID {
		return ErrNotPlayerTurn
	}

	switch action {
	case ActionPlayFromHand:
		p := PlayFromHandPayload{}
		if err := json.Unmarshal(payload, &p); err != nil {
			return err
		}
		if err := g.PlayFromHand(gamePlayer, p.CardIdx, p.BuildPileIdx); err != nil {
			return err
		}
		if len(gamePlayer.Hand) == 0 {
			g.draw(gamePlayer)
		}
		return nil

	case ActionPlayFromDiscard:
		p := PlayFromDiscardPayload{}
		if err := json.Unmarshal(payload, &p); err != nil {
			return err
		}
		return g.PlayFromDiscard(gamePlayer, p.DiscardPileIdx, p.BuildPileIdx)

	case ActionPlayFromStock:
		p := PlayFromStockPayload{}
		if err := json.Unmarshal(payload, &p); err != nil {
			return err
		}
		if err := g.PlayFromStock(gamePlayer, p.BuildPileIdx); err != nil {
			return err
		}
		if len(gamePlayer.StockPile) == 0 {
			player.Score += g.score()
			g = nil
		}
		return nil

	case ActionDiscard:
		p := DiscardPayload{}
		if err := json.Unmarshal(payload, &p); err != nil {
			return err
		}
		if err := g.Discard(gamePlayer, p.CardIdx, p.DiscardPileIdx); err != nil {
			return err
		}

		g.startNextTurn()
		return nil
	default:
		return ErrMsgTypeNotFound
	}
}
