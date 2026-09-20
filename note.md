Il faut créer un mode invité, cela génère un uuid, ce dernier est stocker dans un cookie.
Lorsqu'un user entre dans une room il est prompté de choisir un nom, ce nom est sauvegarder uniquement dans la Room, ou dans le cookie aussi, mais l'idnetifiant reste toujours le uuid.

Lorsqu'un user crée une Room, il en devient le Host. Lorsque le Host quitte la Room, le dernier joueur à avoir rejoint devient le Host.
Si aucun client ne se connecte à un joueur pendant X seconde, le joueur est retirer, si il était host le changement s'effectue.

on unregisterClient()

```go

player := r.Players[playerID]
player.disconnectedAt = timestamp.
time.AfterFunc(30 * time.Second, func() {
    disconnectCheckup <- PlayerID
})

case: disconnectCheckup := <- PlayerID
    player := r.Players[playerID]
    if player == nil {
        return
    }

    if currentTime == player.disconnectedAt + (30 * time.Second) {
        r.unregisterPlayer <- player
    }
```

```

```
