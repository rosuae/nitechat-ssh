package chat

import (
	"fmt"
	"net"

	"golang.org/x/crypto/ssh"
)

type Server struct {
	address string
	config  *ssh.ServerConfig
	hub     *Hub
}

func NewServer(address string, signer ssh.Signer) *Server {
	config := &ssh.ServerConfig{NoClientAuth: true}
	config.AddHostKey(signer)

	return &Server{
		address: address,
		config:  config,
		hub:     NewHub(),
	}
}

func (s *Server) ListenAndServe() error {
	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", s.address, err)
	}
	defer func() {
		if err := listener.Close(); err != nil {
			fmt.Println("failed to close listener:", err)
		}
	}()

	go s.hub.Run()
	fmt.Println("Listening on " + listener.Addr().String())

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	sshConn, channels, requests, err := ssh.NewServerConn(conn, s.config)
	if err != nil {
		fmt.Println("SSH Handshake failed:", err)
		return
	}
	fmt.Printf("New SSH connection from %s (%s)\n", sshConn.RemoteAddr(), sshConn.ClientVersion())

	go ssh.DiscardRequests(requests)
	defer func() {
		if err := sshConn.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	for newChannel := range channels {
		if newChannel.ChannelType() != "session" {
			_ = newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			fmt.Println("Failed to accept the channel:", err)
			return
		}

		client := s.hub.newClient(sshConn.User(), channel)
		go client.handleSession(requests)
	}
}
