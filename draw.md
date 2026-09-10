# Pourquoi `Draw` prend un `*GamePlayer` ?

## La question

```go
func (g *Game) Draw(player *GamePlayer) error
```

Pourquoi un **pointeur** vers le joueur, plutôt qu’un `GamePlayer` par valeur, ou un `playerID string` / `playerIndex int` ?

---

## Réponse courte

`Draw` **modifie la main** du joueur (`player.Hand = append(...)`). En Go, si tu passes une **valeur**, tu travailles sur une **copie** : la main du vrai joueur dans `g.Players` ne change pas.

Un pointeur dit : « mute **cet** joueur-là, celui qui vit dans la partie. »

---

## Valeur vs pointeur ici

```go
// ❌ Par valeur — la main dans g.Players ne bouge pas
func (g *Game) Draw(player GamePlayer) error {
    player.Hand = append(player.Hand, card) // modifie la copie locale
    return nil
}

// ✅ Par pointeur — la mutation est visible partout
func (g *Game) Draw(player *GamePlayer) error {
    player.Hand = append(player.Hand, card)
    return nil
}
```

`GamePlayer` contient un `Hand` (slice). Même avec un receiver valeur, le slice **partage** son tableau sous-jacent — mais **réassigner** `player.Hand` (comme avec `append` qui alloue un nouveau backing array) remplace le slice **dans la copie**, pas dans `g.Players[i]`.

D’où le pointeur : une seule adresse, une seule main à mettre à jour.

---

## Piège dans `TestDraw`

```go
firstPlayer := game.Players[0]   // copie du struct
err := game.Draw(&firstPlayer)   // pointeur vers la copie, pas vers g.Players[0]
```

Ici, `Draw` remplit bien `firstPlayer.Hand`, mais **`game.Players[0].Hand` reste inchangé**.

Appel correct :

```go
err := game.Draw(&game.Players[0])
```

C’est le même pattern que dans `TestDiscard` :

```go
player := &game.Players[0]
game.Discard(player, cardIdx, pileIdx)
```

---

## Et un `playerID string` à la place ?

Possible, mais ce n’est pas la même responsabilité :

```go
func (g *Game) Draw(playerID string) error {
    // 1. retrouver le joueur dans g.Players
    // 2. pioche
    // 3. mettre à jour sa main
}
```

| Approche | Avantage | Inconvénient |
|----------|----------|--------------|
| `*GamePlayer` | Pas de lookup ; l’appelant choisit le joueur ; mutation directe | L’appelant doit passer le **bon** pointeur (`&g.Players[i]`) |
| `playerID` / index | API plus « métier » ; impossible de passer une copie par erreur | `Draw` doit chercher le joueur ; erreur si ID inconnu |

Les deux sont valides. `*GamePlayer` convient bien tant que `Draw`, `Discard`, etc. sont des **actions sur un joueur déjà identifié** par la couche au-dessus (tour en cours, handler WebSocket, etc.).

---

## Règle pratique pour ton projet

- Méthode qui **mute** un joueur ou la partie → receiver `*Game`
- Paramètre qu’on **mute** (`Hand`, `DiscardPiles`, …) → `*GamePlayer`
- Dans les tests → toujours `&game.Players[i]`, jamais `&copieLocale`

`Discard` suit la même logique pour la même raison.
