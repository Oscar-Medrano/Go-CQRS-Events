package main

import (
	"log"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	// hub es una referencia al Hub al que pertenece este cliente
	hub *Hub
	// id es el identificador único del cliente
	id string
	// socket es la conexión WebSocket del cliente
	socket *websocket.Conn
	// outbound es el canal por el que se envían los mensajes al cliente
	outbound chan []byte
}

func NewClient(hub *Hub, socket *websocket.Conn, id string) *Client {
	return &Client{
		hub:      hub,
		socket:   socket,
		id:       id,
		outbound: make(chan []byte),
	}
}

func (c *Client) Write() {
	defer func() {
		log.Printf("Write goroutine exiting for client %s", c.id)
		c.socket.WriteMessage(websocket.CloseMessage, []byte{})
	}()
	log.Printf("Write goroutine started for client %s", c.id)

	// Keep-alive ticker
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case message := <-c.outbound:
			log.Printf("Sending message to client %s: %s", c.id, string(message))
			err := c.socket.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				log.Printf("Error writing message to client %s: %v", c.id, err)
				return
			}
		case <-ticker.C:
			// Send ping to keep connection alive
			err := c.socket.WriteMessage(websocket.PingMessage, []byte("keepalive"))
			if err != nil {
				log.Printf("Error sending ping to client %s: %v", c.id, err)
				return
			}
			log.Printf("Sent ping to client %s", c.id)
		}
	}
}

func (c *Client) Read() {
	defer func() {
		log.Printf("Read goroutine exiting for client %s", c.id)
		c.hub.unregister <- c
	}()
	log.Printf("Read goroutine started for client %s", c.id)

	c.socket.SetPongHandler(func(appData string) error {
		log.Printf("Received pong from client %s", c.id)
		return nil
	})

	for {
		_, message, err := c.socket.ReadMessage()
		if err != nil {
			log.Printf("Error reading message from client %s: %v", c.id, err)
			break
		}
		log.Printf("Received message from client %s: %s", c.id, string(message))

		// Echo back the message
		c.outbound <- message
	}
}
