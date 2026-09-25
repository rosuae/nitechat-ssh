package message

import (
	"fmt"
	"strings"
	"sync"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

type User struct {
	hub         Hub
	ui          *term.Terminal
	channel     ssh.Channel
	username    string
	receive     chan *Message
	closeOnce   sync.Once
	receiveOnce sync.Once
}

type Hub interface {
	RegisterUser(*User)
	UnregisterUser(*User)
	SendMessage(*Message)
}

func NewUser(hub Hub, username string, channel ssh.Channel) *User {
	return &User{
		hub:      hub,
		ui:       term.NewTerminal(channel, "> "),
		channel:  channel,
		username: username,
		receive:  make(chan *Message),
	}
}

func (u *User) Listen() {
	for msg := range u.receive {
		if _, err := u.ui.Write([]byte(msg.String())); err != nil {
			fmt.Println("Failed to write to user:", err)
			return
		}
	}
}

func (u *User) Send(msg *Message) bool {
	select {
	case u.receive <- msg:
		return true
	default:
		return false
	}
}

func (u *User) CloseConn() {
	u.closeOnce.Do(func() {
		if err := u.channel.Close(); err != nil {
			fmt.Println("Failed to close client channel:", err)
		}
	})
}

func (u *User) CloseReceive() {
	u.receiveOnce.Do(func() {
		close(u.receive)
	})
}

func (u *User) Username() string {
	return u.username
}

func (u *User) HandleSession(requests <-chan *ssh.Request) {
	registered := false
	defer func() {
		if registered {
			u.hub.UnregisterUser(u)
		}
		if err := u.channel.Close(); err != nil {
			fmt.Println("Failed to close user channel:", err)
		}
	}()

	go func() {
		for req := range requests {
			if req.Type == "pty-req" || req.Type == "shell" {
				req.Reply(true, nil)
			} else {
				req.Reply(false, nil)
			}
		}
	}()

	if _, err := u.ui.Write([]byte("Welcome to nite-chat!\r\n")); err != nil {
		fmt.Println("Failed to write to user:", err)
		return
	}

	u.hub.RegisterUser(u)
	registered = true
	go u.Listen()

	for {
		line, err := u.ui.ReadLine()
		if err != nil {
			fmt.Println("User disconnected or error:", err)
			return
		}

		line = strings.TrimSpace(line)

		if line == "/leave" {
			return
		}
		msg := NewMessage(u, line)
		u.hub.SendMessage(msg)
	}
}
