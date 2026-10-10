package chat

import (
	"fmt"
	"nite-chat/internal/chat/message"
	"sync"
)

// to do: user hub ownership / allowed users, hub invite
type Hub struct {
	hubName    string
	users      map[*message.User]bool
	broadcast  chan *message.Message
	register   chan *message.User
	unregister chan *message.User
	done       chan struct{}
	onEmpty    func(*Hub)

	mu           sync.RWMutex
	shutdownOnce sync.Once
}

func NewHub(name string, onEmpty ...func(*Hub)) *Hub {
	hub := &Hub{
		hubName:    name,
		users:      make(map[*message.User]bool),
		broadcast:  make(chan *message.Message),
		register:   make(chan *message.User),
		unregister: make(chan *message.User),
		done:       make(chan struct{}),
	}
	if len(onEmpty) > 0 {
		hub.onEmpty = onEmpty[0]
	}
	return hub
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
				h.cleanupIfEmpty()
			}
		case msg := <-h.broadcast:
			for user := range h.users {
				if !user.Send(msg) {
					fmt.Println("Failed to write to user:", user.Username())
					delete(h.users, user)
					user.CloseReceive()
					user.CloseConn()
					h.cleanupIfEmpty()
				}
			}
		}
	}
}

func (h *Hub) Shutdown() {
	h.shutdownOnce.Do(func() {
		close(h.done)
	})
}

func (h *Hub) cleanupIfEmpty() {
	if len(h.users) == 0 && h.onEmpty != nil {
		h.onEmpty(h)
	}
}

func (h *Hub) RegisterUser(user *message.User) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	h.register <- user
	fmt.Println("Register user:", user.Username())
}

func (h *Hub) UnregisterUser(user *message.User) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.unregister <- user
	fmt.Println("Unregister user:", user.Username())
}

func (h *Hub) SendMessage(msg *message.Message) {
	h.broadcast <- msg
}

func (h *Hub) HubDetails() (string, int) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.hubName, len(h.users)
}
