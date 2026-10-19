package chat

import (
	"fmt"
	"nite-chat/internal/chat/message"
	"sync"
	"sync/atomic"
)

const (
	maxUsers = 21
)

type registerRequest struct {
	user   *message.User
	result chan<- bool
}

// to do: user hub ownership / allowed users, hub invite
type Hub struct {
	hubName    string
	users      map[*message.User]bool
	broadcast  chan *message.Message
	register   chan registerRequest
	unregister chan *message.User
	done       chan struct{}
	onEmpty    func(*Hub)
	userCount  atomic.Int64

	shutdownOnce sync.Once
}

func NewHub(name string, onEmpty ...func(*Hub)) *Hub {
	hub := &Hub{
		hubName:    name,
		users:      make(map[*message.User]bool),
		broadcast:  make(chan *message.Message),
		register:   make(chan registerRequest),
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
		case request := <-h.register:
			alreadyConnected := false
			for existingUser := range h.users {
				if existingUser.Username() == request.user.Username() {
					alreadyConnected = true
					break
				}
			}

			if !alreadyConnected && len(h.users) < maxUsers {
				h.users[request.user] = true
				h.userCount.Add(1)
				request.result <- true
			} else {
				request.result <- false
			}
		case user := <-h.unregister:
			if _, ok := h.users[user]; ok {
				delete(h.users, user)
				h.userCount.Add(-1)
				h.cleanupIfEmpty()
			}
		case msg := <-h.broadcast:
			var failed []*message.User
			for user := range h.users {
				if !user.Send(msg) {
					fmt.Println("Failed to write to user:", user.Username())
					failed = append(failed, user)
				}
			}
			for _, user := range failed {
				delete(h.users, user)
				h.userCount.Add(-1)
				user.CloseConn()
			}
			if len(failed) > 0 {
				h.cleanupIfEmpty()
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

func (h *Hub) RegisterUser(user *message.User) bool {
	result := make(chan bool, 1)

	select {
	case h.register <- registerRequest{
		user:   user,
		result: result,
	}:
	case <-h.done:
		return false
	}

	accepted := <-result
	if !accepted {
		return false
	}

	fmt.Println("Register user:", user.Username())
	h.SendMessage(message.NewSystemMessage(user.Username() + " has joined your hub. Say Hi!"))
	return true
}

func (h *Hub) UnregisterUser(user *message.User) {
	select {
	case h.unregister <- user:
	case <-h.done:
		return
	}
	fmt.Println("Unregister user:", user.Username())
	h.SendMessage(message.NewSystemMessage(user.Username() + " has left your hub."))
}

func (h *Hub) SendMessage(msg *message.Message) {
	select {
	case h.broadcast <- msg:
	case <-h.done:
	}
}

// func (h *Hub) unicastSystemMessage(user *message.User, body string) {
// 	if !user.Send(message.NewMessage(nil, body)) {
// 		fmt.Println("Failed to write to user:", user.Username())
// 	}
// }

func (h *Hub) HubDetails() (string, int) {
	return h.hubName, int(h.userCount.Load())
}
