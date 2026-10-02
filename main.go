package main

import (
	"errors"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v5"
)

var upgrader = websocket.Upgrader{}

func main() {
	h := NewHub()
	e := echo.New()

	e.POST("/room/create", func(c *echo.Context) error {
		cookie, err := c.Cookie("id")
		if err != nil {
			return err
		}

		id := PlayerID(cookie.Value)
		if len(id) == 0 {
			return errors.New("invalid id")
		}

		room, err := h.CreateRoom(id)
		if err != nil {
			return err
		}

		go room.run()
		return c.JSON(http.StatusOK, map[string]string{"roomID": string(room.ID)})
	})

	e.GET("/room/:id", func(c *echo.Context) error {
		cookie, err := c.Cookie("id")
		if err != nil {
			return err
		}

		playerID := PlayerID(cookie.Value)
		if playerID == "" {
			return errors.New("invalid id")
		}

		roomID := RoomID(c.Param("id"))
		if roomID == "" {
			return errors.New("invalid roomID")
		}

		wasConnected, err := h.AssignPlayer(playerID, roomID)
		if err != nil {
			return err
		}

		conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			if !wasConnected {
				h.RemovePlayer(playerID)
			}
			return err
		}

		client, err := h.Attach(playerID, conn)
		if err != nil {
			return err
		}

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
