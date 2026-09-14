package main

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

func main() {
	e := echo.New()

	e.POST("/room/create", func(c *echo.Context) error {
		// create roomID
		// create Player
		// create room with roomID, Player
		// upgrade ws
		cookie, err := c.Cookie("id")
		if err != nil {
			return err
		}

		PlayerID := cookie.Value
		return c.String(http.StatusAccepted, PlayerID)
	})

	e.GET("/room/:id/join", func(c *echo.Context) error {
		// get the roomID
		// if found and Player is not already in,
		//	create it.
		//	add it

		// cookie, err := c.Cookie("id")
		// if err != nil {
		//	return err
		// }
		// roomId := c.Param("id")
		// room, err := hub.findRoom(roomID)
		// room.findPlayer(cookie.Id)

		cookie, err := c.Cookie("id")
		if err != nil {
			return err
		}

		PlayerID := cookie.Value
		return c.String(http.StatusAccepted, PlayerID)
	})

	e.POST("/login/:name", func(c *echo.Context) error {
		// name will act as an id.
		// and will be store inside a cookie.
		// **THIS IS TEMPORARY**.
		name := c.Param("name")
		c.SetCookie(&http.Cookie{Name: "id", Value: name})
		return c.String(http.StatusOK, name)
	})
}
