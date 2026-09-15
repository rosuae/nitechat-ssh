// for connecting: ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null localhost -p 2222

package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"net"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

type Client struct {
	ui *term.Terminal
}

type Hub struct {
	clients map[*Client]bool
	broadcast chan []byte
	register chan *Client
	unregister chan *Client
}

func (h Hub) Run() {
	for {
		select {
			case c := <-h.register:
				h.clients[c] = true
			case c := <-h.unregister:
				delete(h.clients, c)
			case message := <-h.broadcast:
				for client := range h.clients {
					client.ui.Write([]byte(string(message[:]) + "\r\n"))
				}
		}
	}
}

func main() {
	startServer()
}

func startServer() {
	mainHub := Hub{}
	go mainHub.Run()
	
	listener, err := net.Listen("tcp", ":2222")
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	config := &ssh.ServerConfig{
		NoClientAuth: true,
	}

	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		fmt.Println("failed to generate key:", err.Error())
		os.Exit(1)
	}

	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		fmt.Println("failed to create signer", err.Error())
		os.Exit(1)
	}

	config.AddHostKey(signer)

	defer listener.Close()
	fmt.Println("Listening on " + listener.Addr().String())

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection", err.Error())
			continue
		}
		go handleConnection(conn, &mainHub, config)
	}
}

func handleConnection(conn net.Conn, hub *Hub, config *ssh.ServerConfig) {
	sshConn, chans, req, err := ssh.NewServerConn(conn, config)
	
	if err != nil {
		fmt.Println("SSH Handshake failed:", err)
		return
	}

	client := Client{}
	hub.register <- &client

	fmt.Printf("New SSH connection from %s (%s)\n", sshConn.RemoteAddr(), sshConn.ClientVersion())

	go ssh.DiscardRequests(req)
	defer func(sshConn *ssh.ServerConn) {
		err := sshConn.Close()
		if err != nil {
			fmt.Println(err.Error())
		}
		hub.unregister <- &client
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

		go HandleSession(channel, request, hub, &client)
	}
}

func HandleSession(chanl ssh.Channel, requests <-chan *ssh.Request, hub *Hub, client *Client) {
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
	}

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
		
		hub.broadcast <- []byte(line + "\r\n")
	}
}
