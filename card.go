package main

import (
	"errors"
	"math/rand"
)

type (
	DiscardPiles [4]Pile
	BuildPiles   [4]Pile
	StockPiles   Pile
	Pile         []Card
	Deck         []Card
	Hand         []Card
	Card         int
)

var SkipBo Card = 0

var (
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

func (d *Deck) Pop() (Card, error) {
	size := len(*d)
	if size == 0 {
		return 0, ErrDeckIsEmpty
	}

	c := (*d)[0]
	*d = (*d)[1:]
	return c, nil
}

func CanPlay(card Card, pile Pile) bool {
	pileSize := len(pile)
	if pileSize >= 12 {
		return false
	}
	nextValue := pileSize + 1
	return nextValue == int(card) || card == SkipBo
}

func validPileIndex(idx int) error {
	if idx < 0 || idx >= 4 {
		return ErrInvalidPileIndex
	}

	return nil
}

func (h Hand) at(idx int) (Card, error) {
	if idx < 0 || idx >= len(h) {
		return 0, ErrInvalidCardIndex
	}

	return h[idx], nil
}
