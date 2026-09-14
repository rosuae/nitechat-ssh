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

func main() {
	startServer()
}

func startServer() {
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
		go handleConnection(conn, config)
	}
}

func handleConnection(conn net.Conn, config *ssh.ServerConfig) {
	sshConn, chans, req, err := ssh.NewServerConn(conn, config)

	if err != nil {
		fmt.Println("SSH Handshake failed:", err)
		return
	}

	fmt.Printf("New SSH connection from %s (%s)\n", sshConn.RemoteAddr(), sshConn.ClientVersion())

	go ssh.DiscardRequests(req)

	for newChannel := range chans {
		if newChannel.ChannelType() != "session"{
			newChannel.Reject(ssh.UnknownChannelType ,"unknown channel type")
			continue
		}

		channel, request, err := newChannel.Accept()
		if err != nil {
			fmt.Println("Failed to accept the channel:", err.Error())
			return
		}

		go HandleSession(channel, request)
	}
	
	sshConn.Close()
}

func HandleSession(chanl ssh.Channel, requests <-chan *ssh.Request) {
	go func() {
		for req := range requests{
			if req.Type == "pty-req" || req.Type == "shell" {
				req.Reply(true, nil)
			} else {
				req.Reply(false, nil)
			}
		}
	}()

	terminal := term.NewTerminal(chanl, "> ")
		
		_, err := terminal.Write([]byte("Welcome to nite-chat!\r\n"))
		if err != nil {
			fmt.Println("Failed to write to user:", err.Error())
		}
	
		for {
			line, err := terminal.ReadLine()
			if err != nil {
				fmt.Println("User disconnected or error:", err.Error())
				return
			}
	
			line = strings.TrimSpace(line)
			
			if line == "/leave" {
				chanl.Close()
				return
			}
	
			terminal.Write([]byte(fmt.Sprintf("You said: %s\r\n", line)))
		}
}
