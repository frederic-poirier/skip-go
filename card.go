package main

import (
	"errors"
	"math/rand"
)

type (
	DiscardPiles [4]Pile
	BuildPiles   [4]Pile
	StockPile    Pile
	Pile         []Card
	Deck         []Card
	Hand         []Card
	Card         int
)

var SkipBo Card = 0

var (
	ErrEmptyPile        = errors.New("pile is empty")
	ErrDeckIsEmpty      = errors.New("deck is empty")
	ErrInvalidPileIndex = errors.New("the pile index provided is out of bound")
	ErrInvalidCardIndex = errors.New("the card index provided is out of bound")
)

func NewDeck() Deck {
	d := Deck{}

	for range 12 {
		for i := range 12 {
			d = append(d, Card(i+1))
		}
	}

	for range 18 {
		d = append(d, SkipBo)
	}

	return d
}

func (d Deck) Shuffle(rng *rand.Rand) {
	rng.Shuffle(len(d), func(i, j int) {
		d[i], d[j] = d[j], d[i]
	})
}

func (d *Deck) Shift() (Card, error) {
	size := len(*d)
	if size == 0 {
		return 0, ErrDeckIsEmpty
	}

	c := (*d)[0]
	*d = (*d)[1:]
	return c, nil
}

func (p *Pile) Pop() error {
	size := len(*p)
	if size == 0 {
		return ErrEmptyPile
	}

	*p = (*p)[:size]
	return nil
}

func (p *Pile) Top() (Card, error) {
	size := len(*p)
	if size == 0 {
		return 0, ErrEmptyPile
	}

	return (*p)[size-1], nil
}

func CanPlay(card Card, pile Pile) bool {
	pileSize := len(pile)
	if pileSize >= 12 {
		return false
	}
	nextValue := pileSize + 1
	return nextValue == int(card) || card == SkipBo
}

func (p Pile) empty() bool {
	return len(p) == 0
}

func (dp DiscardPiles) pile(idx int) (Pile, error) {
	if idx < 0 || idx >= 4 {
		return nil, ErrInvalidPileIndex
	}

	return dp[idx], nil
}

func (bp BuildPiles) pile(idx int) (Pile, error) {
	if idx < 0 || idx >= 4 {
		return nil, ErrInvalidPileIndex
	}

	return bp[idx], nil
}

func (h Hand) at(idx int) (Card, error) {
	if idx < 0 || idx >= len(h) {
		return 0, ErrInvalidCardIndex
	}

	return h[idx], nil
}
