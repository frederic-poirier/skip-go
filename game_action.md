# Answers: `Draw` and `Discard` design questions

Questions from comments in `game_action.go`.

---

## 1. `Draw` — who handles `ErrDeckIsEmpty` from `Pop()`?

**Handle it inside `Draw` (or a private helper it calls), not in the WebSocket router.**

`Draw`'s job is: *refill this player's hand to 5 if possible*. Whether that requires popping from the deck, recycling `CompletedPile`, shuffling, and popping again is **game rule detail** — the caller should not need to know.

```go
func (g *Game) drawCard() (Card, error) {
    card, err := g.Deck.Pop()
    if err == ErrDeckIsEmpty {
        if len(g.CompletedPile) == 0 {
            return 0, ErrNoCardsAvailable // new sentinel: truly stuck
        }
        g.Deck = Deck(g.CompletedPile)
        g.CompletedPile = Pile{}
        g.Deck.Shuffle(g.RNG)
        return g.Deck.Pop()
    }
    return card, err
}
```

### What the caller should see

| Error | Meaning for the router |
|-------|------------------------|
| `ErrDrawHandFull` | Invalid request — hand already at 5 |
| `ErrNoCardsAvailable` | Game cannot continue (edge case) |
| `nil` | Hand refilled as far as rules allow |

The router maps these to WebSocket responses. It should **not** receive raw `ErrDeckIsEmpty` from `Pop()` — that is an internal signal, not a user-facing action failure.

### When would the caller handle it?

Only if `Pop()` were a **public** action the client could invoke directly (“draw one card from deck”). You don't have that — `Pop()` is an implementation detail. Keep empty-deck recovery colocated with the action that needs cards.

---

## 2. Your overall model — is it sound?

Yes. This layering is correct:

```
WebSocket readPump
  → parse + auth (who is this socket?)
  → router (which action? is it this player's turn?)
  → game action (validate rules + mutate state)
  → return result or error
  → broadcast updated state / events to room
```

**Game actions should either apply a valid state change or return an error.** The router should not mutate `Game` itself — it delegates and then broadcasts.

One addition: have actions return something richer than bare `error` when side effects matter (turn change, game over, etc.) — see section 4.

---

## 3. Where should validation live? (e.g. `cardIdx`, `pileIdx`)

Split validation into **layers**, by what each layer knows:

### Router / handler (transport)

- Message is valid JSON
- Required fields present
- Player is connected and identified
- **Optional:** coarse guards (`cardIdx >= 0`, `pileIdx` in `0..3`)

### Game action (domain) — **required**

- `cardIdx` is in range for **this player's current hand**
- `pileIdx` is valid for discard piles
- Hand is not empty
- Any Skip-Bo rule specific to the move

**Rule of thumb:** if the check needs `Game` state, it belongs in the game action (or a helper it calls). The router must not duplicate “is this index valid for this hand right now?” — only `Discard` knows the hand size at call time.

Also: `Discard` currently indexes `player.Hand[cardIdx]` with no bounds check — that will panic on bad input. Validation inside `Discard` (returning `ErrInvalidCardIndex`) protects both WebSocket clients and tests.

### Turn check — router or action?

Either works; pick one and stay consistent:

```go
// Option A: router checks once before any action
if game.CurrentPlayer() != player { return ErrNotYourTurn }

// Option B: shared helper called by every action
func (g *Game) assertCurrentPlayer(p *GamePlayer) error { ... }
```

Prefer **one gate**, not duplicated in every action and the router.

### `Discard` should return `error`

For consistency with `Draw`:

```go
func (g *Game) Discard(player *GamePlayer, cardIdx, pileIdx int) error
```

Same contract: mutate or explain why not.

---

## 4. Turn ending — inside `Discard` or outside?

**Keep `Discard` focused on the discard move. Handle turn transition one level up — but make it explicit so nothing is “lost”.**

In Skip-Bo, a turn is a **sequence**: play cards (optional, multiple) → **discard one card to end**. Only the discard ends the turn. That means:

| Action | Ends turn? |
|--------|------------|
| `PlayFromHand`, `PlayFromStockPile`, `PlayFromDiscardPile` | No |
| `Discard` | Yes |

### Recommended shape

**`Discard` does one thing:**

```go
func (g *Game) Discard(player *GamePlayer, cardIdx, pileIdx int) error {
    // validate, move card hand → discard pile
}
```

**Turn advance is separate:**

```go
func (g *Game) EndTurn() {
    g.TurnIndex = (g.TurnIndex + 1) % len(g.Players)
}

func (g *Game) BeginTurn() error {
    return g.Draw(&g.Players[g.TurnIndex]) // draw to 5 at start of turn
}
```

**The router (or a small orchestrator) wires the rule “discard ends turn”:**

```go
func (g *Game) ExecuteDiscard(player *GamePlayer, cardIdx, pileIdx int) (ActionResult, error) {
    if err := g.Discard(player, cardIdx, pileIdx); err != nil {
        return ActionResult{}, err
    }
    g.EndTurn()
    if err := g.BeginTurn(); err != nil {
        return ActionResult{}, err
    }
    return ActionResult{Events: [...]}, nil
}
```

This avoids burying turn logic inside `Discard` while keeping the Skip-Bo rule in one named place (`ExecuteDiscard` or the router).

### Solving “turn info will be lost”

Don't hide side effects inside `Discard`. Return **events** the hub can broadcast:

```go
type GameEvent struct {
    Type string // "card_discarded", "turn_ended", "turn_started", "hand_drawn"
    // ... payload
}

type ActionResult struct {
    Events []GameEvent
}
```

The WebSocket layer broadcasts each event. Clients learn that a discard happened **and** whose turn it is now — nothing lost, without stuffing turn logic into `Discard`.

### Should the router allow “more than one action”?

Two patterns:

1. **Client sends one command, server runs a scripted sequence** (recommended for discard-end-turn):
   - Client: `{ "action": "discard", "cardIdx": 2, "pileIdx": 1 }`
   - Server: discard → end turn → begin turn → broadcast

2. **Client sends explicit batch** (only if you need it later):
   - `{ "actions": ["discard", "endTurn"] }` — usually unnecessary for Skip-Bo

Prefer **one client intent → one server orchestration**. The router knows “discard implies end turn”; the client does not send two messages.

---

## Summary

| Question | Recommendation |
|----------|----------------|
| `ErrDeckIsEmpty` in `Draw`? | Handle inside `Draw` via recycle + shuffle; don't leak to router |
| Action = mutate or error? | Yes — good model; add `ActionResult` with events for side effects |
| Index validation where? | Domain rules inside the action; router does transport/auth only |
| Turn end in `Discard`? | No — `Discard` moves a card; orchestrator/router runs discard → `EndTurn` → `BeginTurn` |
| Lost turn information? | Return events from the orchestrated call; broadcast them over WebSocket |

Your instinct that game actions live in the domain layer is right. The refinement: **small pure actions** (`Discard`, `Draw`) + **thin orchestration** for sequences that span multiple steps (`ExecuteDiscard`, turn lifecycle).
