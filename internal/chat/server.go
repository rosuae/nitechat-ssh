package chat

import (
	"errors"
	"fmt"
	"net"
	"nite-chat/internal/chat/message"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

type Server struct {
	address string
	config  *ssh.ServerConfig
	hub     *Hub

	allowlistOnly  bool
	keysMu         sync.RWMutex
	authorizedKeys map[string]struct{} // turn into ssh.PublicKey

	mu       sync.Mutex
	listener net.Listener
}

func NewServer(address string, signer ssh.Signer, allowlistOnly bool) *Server {
	s := &Server{
		address:        address,
		hub:            NewHub(),
		allowlistOnly:  allowlistOnly,
		authorizedKeys: make(map[string]struct{}),
	}
	s.config = &ssh.ServerConfig{
		MaxAuthTries: 3,
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			fp := ssh.FingerprintSHA256(key)
			if s.allowlistOnly && !s.isAuthorizedKey(fp) {
				return nil, fmt.Errorf("%s is not authorized", fp)
			}
			return &ssh.Permissions{
				Extensions: map[string]string{"fingerprint": fp},
			}, nil
		},
	}
	s.config.AddHostKey(signer)
	return s
}

func (s *Server) isAuthorizedKey(key string) bool {
	s.keysMu.RLock()
	defer s.keysMu.RUnlock()
	_, ok := s.authorizedKeys[key]
	return ok
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
	conn.SetDeadline(time.Now().Add(30 * time.Second))

	sshConn, channels, requests, err := ssh.NewServerConn(conn, s.config)
	if err != nil {
		fmt.Println("SSH Handshake failed:", err)
		conn.Close()
		return
	}
	fp := sshConn.Permissions.Extensions["fingerprint"]

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

		user := message.NewUser(s.hub, fp, sshConn.User(), channel)
		go user.HandleSession(requests)
	}
}
