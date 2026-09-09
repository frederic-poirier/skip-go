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

var ErrDeckIsEmpty = errors.New("deck is empty")

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
