// for connecting: ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null localhost -p 2222

package main

import (
	"fmt"
	"net"
	"os"
	"strings"

	"crypto/ed25519"
	"crypto/rand"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

type Client struct {
	ui       *term.Terminal
	id       string
	username string
	receive  chan []byte
}

func (c *Client) Listen() {
	for message := range c.receive {
		if _, err := c.ui.Write(message); err != nil {
			fmt.Println("Failed to write to user:", err.Error())
			return
		}
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
				client.receive <- message
			}
		}
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

	privateKeyBytes, err := os.ReadFile(".ssh/host_ed25519")
	if err != nil {
		_, privateKey, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			fmt.Println("failed to generate key:", err.Error())
			os.Exit(1)
		}
		if err = os.WriteFile(".ssh/host_ed25519", privateKey, 0600); err != nil {
			fmt.Println("failed to write ed25519 key:", err.Error())
			os.Exit(1)
		}
		privateKeyBytes = privateKey
	}

	privateKey, err := ssh.ParseRawPrivateKey(privateKeyBytes)
	if err != nil {
		fmt.Println("failed to parse private key:", err.Error())
	}

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
		go handleConnection(conn, mainHub, config)
	}
}

func handleConnection(conn net.Conn, hub *Hub, config *ssh.ServerConfig) {
	sshConn, chans, req, err := ssh.NewServerConn(conn, config)

	if err != nil {
		fmt.Println("SSH Handshake failed:", err)
		return
	}

	fmt.Printf("New SSH connection from %s (%s)\n", sshConn.RemoteAddr(), sshConn.ClientVersion())

	go ssh.DiscardRequests(req)
	defer func(sshConn *ssh.ServerConn) {
		err := sshConn.Close()
		if err != nil {
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

		client := &Client{
			username: sshConn.User(),
			receive:  make(chan []byte, 16),
		}
		go HandleSession(channel, request, hub, client)
	}
}

func HandleSession(chanl ssh.Channel, requests <-chan *ssh.Request, hub *Hub, client *Client) {
	registered := false
	defer func() {
		if registered {
			hub.unregisterClient(client)
		}
		if err := chanl.Close(); err != nil {
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

	client.ui = term.NewTerminal(chanl, "> ")

	_, err := client.ui.Write([]byte("Welcome to nite-chat!\r\n"))
	if err != nil {
		fmt.Println("Failed to write to user:", err.Error())
		return
	}

	hub.registerClient(client)
	registered = true
	go client.Listen()

	for {
		line, err := client.ui.ReadLine()
		if err != nil {
			fmt.Println("User disconnected or error:", err.Error())
			return
		}

		line = strings.TrimSpace(line)

		if line == "/leave" {
			chanl.Close()
			return
		}

		hub.sendMessage([]byte(line + "\r\n"))
	}
}
