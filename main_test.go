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
				StockPile{},
				DiscardPiles{},
			}, {
				"2",
				Hand{},
				StockPile{},
				DiscardPiles{},
			},
		},
	}

	game.Deck.Shuffle(game.RNG)
	game.deal(StockPilesSize)

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
	playerOne := GamePlayer{"0", Hand{1, 2, 3}, StockPile{}, DiscardPiles{}}
	playerTwo := GamePlayer{"1", Hand{1, 2, 3, 4, 5}, StockPile{}, DiscardPiles{}}
	deck := Deck{1, 2, 3, 4, 5}

	game := &Game{
		Deck:    deck,
		Players: []GamePlayer{playerOne, playerTwo},
	}

	firstPlayer := game.Players[0]
	game.draw(&firstPlayer)

	if len(firstPlayer.Hand) != 5 {
		t.Errorf("la main doit posséder 5 cartes après l'invocation de draw, reçu %v ", len(firstPlayer.Hand))
	}

	if len(game.Deck) != 3 {
		t.Errorf("deck size is not matching %v", len(game.Deck))
	}
}

func TestCanPlay(t *testing.T) {
	t.Run("jouer sur une pile vide", func(t *testing.T) {
		card := Card(1)
		pile := Pile{}
		if !CanPlay(card, pile) {
			t.Errorf("CanPlay devrait accepter la carte 1 ou SkipBo sur un pile empty")
		}
	})

	t.Run("jouer sur une pile complète", func(t *testing.T) {
		card := Card(12)
		pile := Pile{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
		if CanPlay(card, pile) {
			t.Errorf("CanPlay ne devrait pas accepter de carte sur une pile pleine")
		}
	})

	t.Run("jouer une carte inférieur", func(t *testing.T) {
		card := Card(5)
		pile := Pile{1, 2, 3, 4, 5}
		if CanPlay(card, pile) {
			t.Errorf("CanPlay ne devrait pas accepter une carte inférieur à la dernière carte de la pile")
		}
	})

	t.Run("jouer une carte supérieur", func(t *testing.T) {
		card := Card(4)
		pile := Pile{1, 2}
		if CanPlay(card, pile) {
			t.Error("CanPlay ne devrait pas accepter de carte supérieur à la dernière carte de la pile")
		}
	})

	t.Run("jouer un skipbo", func(t *testing.T) {
		card := SkipBo
		pile := Pile{0, 0, 3, 4}
		if !CanPlay(card, pile) {
			t.Error("CanPlay devrait accepter SkipBo en tout temps excepter lorsque la pile est pleine.")
		}
	})
}

func TestDiscard(t *testing.T) {
	players := []GamePlayer{{
		ID:           "0",
		Hand:         Hand{2, 5, 3, 0, 12},
		DiscardPiles: DiscardPiles{},
	}, {
		ID:           "1",
		Hand:         Hand{4, 2, 3},
		DiscardPiles: DiscardPiles{},
	}}

	game := &Game{
		TurnIndex: 0,
		Deck:      Deck{},
		Players:   players,
	}

	player := &game.Players[0]
	cardIdx := 2
	pileIdx := 2

	game.Discard(player, cardIdx, pileIdx)

	if len(player.Hand) != 4 {
		t.Error("discard devrait retirer une carte de la main du joueur")
	}

	if player.DiscardPiles[pileIdx][0] != 3 {
		t.Error("discard devrait déplacer la carte de la main vers la DiscardPile indiquer")
	}
}

func TestStartNextTurn(t *testing.T) {
	players := []GamePlayer{{
		Hand: Hand{2, 5, 3, 0, 12},
	}, {
		Hand: Hand{4, 2, 3},
	}}

	game := &Game{
		TurnIndex: 0,
		Deck:      Deck{1, 2, 3, 4, 5},
		Players:   players,
	}

	game.startNextTurn()

	if game.TurnIndex == 0 {
		t.Error("TurnIndex devrait changer")
	}

	if len(game.Players[1].Hand) != 5 {
		t.Error("The next player should have 5 cards in is hand.")
	}
}

func TestPlayFromHand(t *testing.T) {
	player := []GamePlayer{{
		Hand: Hand{2, 5, 3, 0, 12},
	}}

	game := &Game{
		Players:    player,
		BuildPiles: BuildPiles{},
	}

	cardIdx := 3
	pileIdx := 2
	err := game.PlayFromHand(&game.Players[0], cardIdx, pileIdx)
	if err != nil {
		t.Errorf("playFromHand n'aurait pas du déclencher d'erreur, reçu: %v", err)
	}
}

func TestPlayFromStock(t *testing.T) {
	game := &Game{
		Players:    []GamePlayer{{StockPile: StockPile{1, 0, 5, 2, 7}}},
		BuildPiles: BuildPiles{Pile{}, Pile{1, 2, 3, 4}},
	}

	pileIdx := 0
	err := game.PlayFromStock(&game.Players[0], pileIdx)
	if err != nil {
		t.Errorf("playFromStock ne devrait pas retourner d'erreur, reçu: %v", err)
	}

	if game.BuildPiles[pileIdx][0] != Card(1) {
		t.Error("la BuildPile devrait avoir la première carte de la stockpile")
	}

	pileIdx = 1
	err = game.PlayFromStock(&game.Players[0], pileIdx)
	if err != nil {
		t.Errorf("playFromStock ne devrait pas retourner d'erreur, reçu: %v", err)
	}

	if len(game.BuildPiles[pileIdx]) != 5 {
		t.Error("la build pile devrait contenir 5 cartes")
	}

	if game.BuildPiles[pileIdx][len(game.BuildPiles[pileIdx])-1] != Card(0) {
		t.Error("la build pile devrait contenir un SkipBo a la fin")
	}

	err = game.PlayFromStock(&game.Players[0], pileIdx)
	if err == nil {
		t.Errorf("playFromStock devrait retourner une erreur car le mouvement est illegale. \nbuildpile: %v \nstockpile: %v", game.BuildPiles[pileIdx], game.Players[0].StockPile)
	}
}

func TestPlayFromDiscard(t *testing.T) {
	game := &Game{
		Players: []GamePlayer{{DiscardPiles: DiscardPiles{
			{0, 3, 5},
			{},
		}}},
		BuildPiles: BuildPiles{
			{1, 2, 3, 4},
			{},
		},
	}

	dpIdx := 0
	bpIdx := 0
	err := game.PlayFromDiscard(&game.Players[0], dpIdx, bpIdx)
	if err != nil {
		t.Errorf("play-from-discard ne devrait pas déclencher d'erreur, reçu: %v", err)
	}

	if game.BuildPiles[bpIdx][len(game.BuildPiles[bpIdx])-1] != 5 {
		t.Errorf("la build pile devrait avoir la dernière carte de la discard pile sélectionner")
	}

	bpIdx = 1
	err = game.PlayFromDiscard(&game.Players[0], dpIdx, bpIdx)
	if err != ErrIllegalMove {
		t.Error("play-from-discard ne devrait pas accepter une carte 3 sur une empty build pile")
	}

	dpIdx = 1
	err = game.PlayFromDiscard(&game.Players[0], dpIdx, bpIdx)
	if err != ErrIllegalMove {
		t.Error("play-from-discard ne devrait pas accepter l'action sur une empty discard pile")
	}
}
