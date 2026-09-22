package chat

import (
	"fmt"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.receive)
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				if !client.send(message) {
					fmt.Println("Failed to write to user:", client.username)
					delete(h.clients, client)
					close(client.receive)
					client.closeConn()
				}
			}
		}
	}
}

func (h *Hub) newClient(username string, channel ssh.Channel) *Client {
	return &Client{
		hub:      h,
		ui:       term.NewTerminal(channel, "> "),
		channel:  channel,
		username: username,
		receive:  make(chan []byte),
	}
}

func (h *Hub) registerClient(client *Client) {
	h.register <- client
	fmt.Println("Register client:", client.username)
}

func (h *Hub) unregisterClient(client *Client) {
	h.unregister <- client
	fmt.Println("Unregister client:", client.username)
}

func (h *Hub) sendMessage(message []byte) {
	h.broadcast <- message
}
