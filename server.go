package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"crypto/ed25519"
	"crypto/rand"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

type Client struct {
	hub       *Hub
	ui        *term.Terminal
	channel   ssh.Channel
	id        string
	username  string
	receive   chan []byte
	closeOnce sync.Once
}

func (c *Client) Listen() {
	for message := range c.receive {
		if _, err := c.ui.Write(message); err != nil {
			fmt.Println("Failed to write to user:", err.Error())
			return
		}
	}
}

func (c *Client) Send(message []byte) bool {
	select {
	case c.receive <- message:
		return true
	default:
		return false
	}
}

func (c *Client) CloseConn() {
	c.closeOnce.Do(func() {
		if err := c.channel.Close(); err != nil {
			fmt.Println("Failed to close client channel:", err.Error())
		}
	})
}

func (c *Client) HandleSession(requests <-chan *ssh.Request, hub *Hub) {
	registered := false
	defer func() {
		if registered {
			hub.unregisterClient(c)
		}
		if err := c.channel.Close(); err != nil {
			fmt.Println("Failed to close user channel:", err.Error())
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

	_, err := c.ui.Write([]byte("Welcome to nite-chat!\r\n"))
	if err != nil {
		fmt.Println("Failed to write to user:", err.Error())
		return
	}

	c.hub.registerClient(c)
	registered = true
	go c.Listen()

	for {
		line, err := c.ui.ReadLine()
		if err != nil {
			fmt.Println("User disconnected or error:", err.Error())
			return
		}
		line = strings.TrimSpace(line)
		if line == "/leave" {
			return
		}
		hub.sendMessage([]byte(line + "\r\n"))
	}
}

type Hub struct {
	id         string
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
		case c := <-h.register:
			h.clients[c] = true
		case c := <-h.unregister:
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.receive)
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				if !client.Send(message) {
					fmt.Println("Failed to write to user:", client.username)
					delete(h.clients, client)
					close(client.receive)
					client.CloseConn()
				}
			}
		}
	}
}

func (h *Hub) newClient(username string, sshChanl ssh.Channel) *Client {
	return &Client{
		hub:      h,
		ui:       term.NewTerminal(sshChanl, "> "),
		channel:  sshChanl,
		username: username,
		receive:  make(chan []byte),
	}
}

func (h *Hub) registerClient(c *Client) {
	h.register <- c
	fmt.Println("Register client:", c.username, ", into hub: ", h.id)
}

func (h *Hub) unregisterClient(c *Client) {
	h.unregister <- c
	fmt.Println("Unregister client:", c.username, ", from hub: ", h.id)
}

func (h *Hub) sendMessage(message []byte) {
	h.broadcast <- message
}

func (h *Hub) handleConnection(conn net.Conn, config *ssh.ServerConfig) {
	sshConn, chans, req, err := ssh.NewServerConn(conn, config)

	if err != nil {
		fmt.Println("SSH Handshake failed:", err)
		return
	}
	fmt.Printf("New SSH connection from %s (%s)\n", sshConn.RemoteAddr(), sshConn.ClientVersion())

	go ssh.DiscardRequests(req)
	defer func(sshConn *ssh.ServerConn) {
		if err := sshConn.Close(); err != nil {
			fmt.Println(err.Error())
		}
	}(sshConn)

	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, request, err := newChannel.Accept()
		if err != nil {
			fmt.Println("Failed to accept the channel:", err.Error())
			return
		}

		client := h.newClient(sshConn.User(), channel)
		go client.HandleSession(request, h)
	}
}

func main() {
	startServer()
}

func startServer() {
	mainHub := NewHub()
	go mainHub.Run()

	listener, err := net.Listen("tcp", ":2222")
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	config := &ssh.ServerConfig{
		NoClientAuth: true,
	}

	// to do
	hostKeyPath := filepath.Join(".ssh", "host_ed25519")
	privateKeyBytes, err := os.ReadFile(hostKeyPath)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Println("failed to read host key:", err.Error())
			os.Exit(1)
		}

		if err := os.MkdirAll(filepath.Dir(hostKeyPath), 0700); err != nil {
			fmt.Println("failed to create host key directory:", err.Error())
			os.Exit(1)
		}

		_, privateKey, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			fmt.Println("failed to generate key:", err.Error())
			os.Exit(1)
		}

		privateKeyBytes = privateKey
		if err = os.WriteFile(hostKeyPath, privateKeyBytes, 0600); err != nil {
			fmt.Println("failed to write ed25519 key:", err.Error())
			os.Exit(1)
		}
	}

	if len(privateKeyBytes) != ed25519.PrivateKeySize {
		fmt.Println("failed to load host key: invalid Ed25519 private key size")
		os.Exit(1)
	}

	privateKey := ed25519.PrivateKey(privateKeyBytes)
	// to do
	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		fmt.Println("failed to create signer", err.Error())
		os.Exit(1)
	}
	config.AddHostKey(signer)

	defer func(listener net.Listener) {
		err := listener.Close()
		if err != nil {
			fmt.Println("failed to close listener:", err.Error())
		}
	}(listener)
	fmt.Println("Listening on " + listener.Addr().String())

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection", err.Error())
			continue
		}
		go mainHub.handleConnection(conn, config)
	}
}
