package main

import (
	"fmt"
	"net"
	"os"
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

	defer listener.Close()
	fmt.Println("Listening on " + listener.Addr().String())

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection", err.Error())
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		fmt.Println("Error reading", err.Error())
		return
	}

	fmt.Println("Received connection", string(buf[:n]))
	conn.Write([]byte("Hello from Go server!\n"))
}
