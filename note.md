Couche:
Requête HTTP JOIN
Information: RoomID, PlayerID

```go
err := h.registerPlayer(PlayerID, RoomID)
if err != nil {
    return c.text(http.StatusCodeError, err.Error())
}

// else the player have joined
```
