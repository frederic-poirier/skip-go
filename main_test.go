package main

import (
	"math/rand"
	"slices"
	"testing"
	"time"
)

func TestNewDeck(t *testing.T) {
	resultat := NewDeck()
	attendu := &Deck{
		1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12,
		1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12,
		1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12,
		1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12,
		1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12,
		1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12,
		1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12,
		1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12,
		1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12,
		1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12,
		1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12,
		1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12,
		0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0,
	}

	if !slices.Equal(resultat, *attendu) {
		t.Errorf("reçu %v attendu %v", resultat, attendu)
	}
}

func TestNewGame(t *testing.T) {
	t.Run("New Game: enough player", func(t *testing.T) {
		players := []Player{
			{"1", 0, "fred"},
			{"0", 0, "em"},
		}

		resultat, err := NewGame(players, time.Now().UnixNano())
		if err != nil {
			t.Fatalf("erreur inattendue: %v", err)
		}

		if len(resultat.Players) != 2 {
			t.Errorf("reçu %d joueurs, attendu 2", len(resultat.Players))
		}

		if len(resultat.Deck) != 162 {
			t.Errorf("reçu un deck de %d cartes, attendu 162", len(resultat.Deck))
		}
	})

	t.Run("New Game: not enough player", func(t *testing.T) {
		players := []Player{
			{"1", 0, "fred"},
		}

		_, err := NewGame(players, time.Now().UnixNano())
		attendu := ErrNotEnoughPlayer

		if err != attendu {
			t.Errorf("reçu %v attendu %v", err, attendu)
		}
	})

	t.Run("New Game: too much player", func(t *testing.T) {
		players := []Player{
			{"0", 0, "fred"},
			{"1", 0, "em"},
			{"2", 0, "derf"},
			{"3", 0, "me"},
			{"4", 0, "dref"},
			{"5", 0, "redf"},
			{"6", 0, "fedr"},
		}

		_, err := NewGame(players, time.Now().UnixNano())
		attendu := ErrTooMuchPlayer

		if err != attendu {
			t.Errorf("reçu %v attendu %v", err, attendu)
		}
	})
}

// [todo] this probably does not test it right or does not need a test.
func TestShuffle(t *testing.T) {
	game := &Game{
		RNG:  rand.New(rand.NewSource(time.Now().UnixNano())),
		Deck: NewDeck(),
	}

	game.Deck.Shuffle(game.RNG)

	if slices.Equal(game.Deck, NewDeck()) {
		t.Errorf("shuffle n'as pas mélanger le deck")
	}
}

func TestDeal(t *testing.T) {
	StockPilesSize := 15
	game := &Game{
		RNG:  rand.New(rand.NewSource(time.Now().UnixNano())),
		Deck: NewDeck(),
		Players: []GamePlayer{
			{
				"1",
				Hand{},
				StockPiles{},
				DiscardPiles{},
			}, {
				"2",
				Hand{},
				StockPiles{},
				DiscardPiles{},
			},
		},
	}

	game.Deck.Shuffle(game.RNG)
	game.Deal(StockPilesSize)

	for _, player := range game.Players {
		if len(player.Hand) != 5 {
			t.Errorf("tout joueur doit avoir 5 cartes dans sa main après le deal. valeur reçu: %v", len(player.Hand))
		}

		if len(player.StockPile) != StockPilesSize {
			t.Errorf("la stockpile ne contient pas la taille spécifier, valeur reçu: %v", len(player.StockPile))
		}
	}

	cardsDealt := (StockPilesSize + 5) * len(game.Players)
	cardsTotal := 12*12 + 18
	if len(game.Deck) > cardsTotal-cardsDealt {
		t.Errorf("pas assez de cartes ont été distribué")
	} else if len(game.Deck) < cardsTotal-cardsDealt {
		t.Errorf("trop de cartes ont été distribué \n nombre de carte dans le paquet: %v\n nombre de carte donner: %v\n nombre de carte attendu: %v", len(game.Deck), cardsTotal-len(game.Deck), cardsDealt)
	}
}

func TestDraw(t *testing.T) {
	playerOne := GamePlayer{"0", Hand{1, 2, 3}, StockPiles{}, DiscardPiles{}}
	playerTwo := GamePlayer{"1", Hand{1, 2, 3, 4, 5}, StockPiles{}, DiscardPiles{}}
	deck := Deck{1, 2, 3, 4, 5}

	game := &Game{
		Deck:    deck,
		Players: []GamePlayer{playerOne, playerTwo},
	}
	firstPlayer := game.Players[0]

	err := game.Draw(&firstPlayer)
	if err != nil {
		t.Errorf("reçu %v mais attendais aucune erreur", err)
	}

	if len(firstPlayer.Hand) != 5 {
		t.Errorf("la main doit posséder 5 cartes après l'invocation de draw, reçu %v ", len(firstPlayer.Hand))
	}

	if len(game.Deck) != 3 {
		t.Errorf("deck size is not matching %v", len(game.Deck))
	}

	t.Log(firstPlayer.Hand, game.Deck)
}
