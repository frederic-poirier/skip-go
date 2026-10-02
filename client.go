package main

import (
	"errors"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	conn     *websocket.Conn
	send     chan ServerMessage
	done     chan struct{}
	playerID PlayerID
	room     *Room
	once     sync.Once
}

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
)

var (
	ERR_CLIENT_BUFFER_FULL = errors.New("could not send the message, the client buffer is full")
	ERR_CLIENT_CLOSE       = errors.New("could not send the message, the client is close")
)

func NewClient(conn *websocket.Conn, playerID PlayerID, room *Room) *Client {
	return &Client{
		conn:     conn,
		send:     make(chan ServerMessage, 64),
		done:     make(chan struct{}),
		playerID: playerID,
		room:     room,
	}
}

func (c *Client) shutdown() {
	c.once.Do(func() {
		close(c.send)
		close(c.done)
		c.conn.Close()
	})
}

func (c *Client) trySend(msg ServerMessage) error {
	select {
	case <-c.done:
		return ERR_CLIENT_CLOSE
	default:
	}

	select {
	case c.send <- msg:
		return nil
	default:
		return ERR_CLIENT_BUFFER_FULL
	}
}

func (c *Client) readPump() {
	defer func() {
		c.room.unregisterClient <- c
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		var msg ClientMessage
		if err := c.conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Printf("erreur inattendue: %v", err)
			}
			break
		}
		err := c.room.send(Command{PlayerID: c.playerID, Message: msg})
		if err != nil {
			log.Printf("erreur lors de l'envoie de la commande: %v", err)
			break
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.shutdown()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteJSON(msg); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
