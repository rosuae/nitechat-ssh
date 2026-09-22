package chat

import (
	"fmt"
	"strings"
	"sync"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

type Client struct {
	hub       *Hub
	ui        *term.Terminal
	channel   ssh.Channel
	username  string
	receive   chan []byte
	closeOnce sync.Once
}

func (c *Client) listen() {
	for message := range c.receive {
		if _, err := c.ui.Write(message); err != nil {
			fmt.Println("Failed to write to user:", err)
			return
		}
	}
}

func (c *Client) send(message []byte) bool {
	select {
	case c.receive <- message:
		return true
	default:
		return false
	}
}

func (c *Client) closeConn() {
	c.closeOnce.Do(func() {
		if err := c.channel.Close(); err != nil {
			fmt.Println("Failed to close client channel:", err)
		}
	})
}

func (c *Client) handleSession(requests <-chan *ssh.Request) {
	registered := false
	defer func() {
		if registered {
			c.hub.unregisterClient(c)
		}
		if err := c.channel.Close(); err != nil {
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

	if _, err := c.ui.Write([]byte("Welcome to nite-chat!\r\n")); err != nil {
		fmt.Println("Failed to write to user:", err)
		return
	}

	c.hub.registerClient(c)
	registered = true
	go c.listen()

	for {
		line, err := c.ui.ReadLine()
		if err != nil {
			fmt.Println("User disconnected or error:", err)
			return
		}

		line = strings.TrimSpace(line)
		if line == "/leave" {
			return
		}
		c.hub.sendMessage([]byte(line + "\r\n"))
	}
}
