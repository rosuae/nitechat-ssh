package chat

import (
	"fmt"
	"nite-chat/internal/chat/message"
)

type Hub struct {
	users      map[*message.User]bool
	broadcast  chan *message.Message
	register   chan *message.User
	unregister chan *message.User
	done       chan struct{}
}

func NewHub() *Hub {
	return &Hub{
		users:      make(map[*message.User]bool),
		broadcast:  make(chan *message.Message),
		register:   make(chan *message.User),
		unregister: make(chan *message.User),
		done:       make(chan struct{}),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case <-h.done:
			return
		case user := <-h.register:
			h.users[user] = true
		case user := <-h.unregister:
			if _, ok := h.users[user]; ok {
				delete(h.users, user)
				user.CloseReceive()
			}
		case msg := <-h.broadcast:
			for user := range h.users {
				if !user.Send(msg) {
					fmt.Println("Failed to write to user:", user.Username())
					delete(h.users, user)
					user.CloseReceive()
					user.CloseConn()
				}
			}
		}
	}
}

func (h *Hub) Shutdown() {
	close(h.done)
}

func (h *Hub) RegisterUser(user *message.User) {
	h.register <- user
	fmt.Println("Register user:", user.Username())
}

func (h *Hub) UnregisterUser(user *message.User) {
	h.unregister <- user
	fmt.Println("Unregister user:", user.Username())
}

func (h *Hub) SendMessage(msg *message.Message) {
	h.broadcast <- msg
}
