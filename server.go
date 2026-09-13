// for connecting: ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null localhost -p 2222

package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"net"
	"os"

	"golang.org/x/crypto/ssh"
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
	sshConn, _, _, err := ssh.NewServerConn(conn, config)

	if err != nil {
		fmt.Println("SSH Handshake failed:", err)
		return
	}

	fmt.Printf("New SSH connection from %s (%s)\n", sshConn.RemoteAddr(), sshConn.ClientVersion())
	sshConn.Close()
}
