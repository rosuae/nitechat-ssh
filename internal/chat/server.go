package chat

import (
	"errors"
	"fmt"
	"net"
	"nite-chat/internal/chat/message"
	"sync"

	"golang.org/x/crypto/ssh"
)

type Server struct {
	address string
	config  *ssh.ServerConfig
	hub     *Hub

	mu       sync.Mutex
	listener net.Listener
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

func (s *Server) Shutdown() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	listener := s.listener

	if listener == nil {
		return nil
	}
	return listener.Close()
}

func (s *Server) ListenAndServe() error {
	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", s.address, err)
	}

	s.mu.Lock()
	s.listener = listener
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.listener = nil
		s.mu.Unlock()

		_ = listener.Close()
	}()

	go s.hub.Run()
	fmt.Println("Listening on " + listener.Addr().String())

	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			return fmt.Errorf("accept connection: %w", err)
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

		user := message.NewUser(s.hub, sshConn.User(), channel)
		go user.HandleSession(requests)
	}
}
