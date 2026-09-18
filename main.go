package main

import (
	"errors"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v5"
)

var upgrader = websocket.Upgrader{}

func main() {
	h := NewHub()
	go h.run()

	e := echo.New()

	e.POST("/room/create", func(c *echo.Context) error {
		cookie, err := c.Cookie("id")
		if err != nil {
			return err
		}

		PlayerID := cookie.Value
		if len(PlayerID) == 0 {
			return errors.New("invalid id")
		}

		room := h.NewRoom()
		log.Printf("new room running")
		go room.run()

		return c.JSON(http.StatusOK, map[string]string{"roomID": room.ID})
	})

	e.GET("/room/:id", func(c *echo.Context) error {
		cookie, err := c.Cookie("id")
		if err != nil {
			return err
		}

		playerID := cookie.Value
		if playerID == "" {
			return errors.New("invalid id")
		}

		roomID := c.Param("id")
		if roomID == "" {
			return errors.New("invalid roomID")
		}

		room := h.getRoom(roomID)
		if room == nil {
			return errors.New("could not find the room")
		}

		conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			return err
		}

		log.Printf("New connection %v in room %v", playerID, roomID)
		client := room.NewClient(playerID, conn)
		go client.readPump()
		go client.writePump()

		return nil
	})

	e.POST("/login/:name", func(c *echo.Context) error {
		// name will act as an id.
		// and will be store inside a cookie.
		// **THIS IS TEMPORARY**.
		name := c.Param("name")
		c.SetCookie(&http.Cookie{Name: "id", Value: name, Path: "/"})
		return c.String(http.StatusOK, name)
	})

	if err := e.Start(":1500"); err != nil {
		e.Logger.Error("shutting down the server", "error", err)
	}
}
