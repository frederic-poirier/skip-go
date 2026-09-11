package main

import (
	"encoding/json"
	"errors"
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
	Ready bool
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

// client request /room/create
// - create roomID
// - create room
// - add player to the room

// client request /room/:id/join
// - add player to the room
//
//

var (
	ErrNotPlayerTurn   = errors.New("is not player's turn")
	ErrMsgTypeNotFound = errors.New("msg type could not be found")
)

func GameActionRouter(g *Game, player Player, e Enveloppe) error {
	gamePlayer := g.findPlayerTurn()
	if gamePlayer.ID != player.ID {
		return ErrNotPlayerTurn
	}

	switch e.Type {
	case MsgTypePlayHand:
		p := PlayFromHandPayload{}
		if err := json.Unmarshal(e.Payload, p); err != nil {
			return err
		}
		if err := g.PlayFromHand(gamePlayer, p.CardIdx, p.BuildPileIdx); err != nil {
			return err
		}
		if len(gamePlayer.Hand) == 0 {
			g.draw(gamePlayer)
		}
		return nil

	case MsgTypePlayDiscard:
		p := PlayFromDiscardPayload{}
		if err := json.Unmarshal(e.Payload, p); err != nil {
			return err
		}
		return g.PlayFromDiscard(gamePlayer, p.DiscardPileIdx, p.BuildPileIdx)

	case MsgTypePlayStock:
		p := PlayFromStockPayload{}
		if err := json.Unmarshal(e.Payload, p); err != nil {
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

	case MsgTypeDiscard:
		p := DiscardPayload{}
		if err := json.Unmarshal(e.Payload, p); err != nil {
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
