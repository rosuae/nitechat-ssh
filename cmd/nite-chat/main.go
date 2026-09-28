package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"nite-chat/internal/chat"
	"nite-chat/internal/hostkey"
)

func main() {
	hostKeyPath, err := hostkey.DefaultPath()
	if err != nil {
		fmt.Println("failed to find host key directory:", err)
		os.Exit(1)
	}

	signer, err := hostkey.LoadSigner(hostKeyPath)
	if err != nil {
		fmt.Println("failed to load host key:", err)
		os.Exit(1)
	}

	server := chat.NewServer(":2222", signer, false)

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.ListenAndServe()
	}()

	fmt.Println("/quit to stop the server.")

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		command := strings.TrimSpace(scanner.Text())

		switch command {
		case "/quit":
			if err := server.Shutdown(); err != nil {
				fmt.Println("failed to shutdown server: ", err)
				os.Exit(1)
			}
			if err := <-serverErr; err != nil {
				fmt.Println("server stopped with error: ", err)
				os.Exit(1)
			}
			fmt.Println("server stopped successfully")
			return
		case "":
			continue
		default:
			fmt.Println("unknown command:", command)
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Println("failed to read terminal command:", err)
	}

	_ = server.Shutdown()

	if err := <-serverErr; err != nil {
		fmt.Println("server stopped with error:", err)
		os.Exit(1)
	}
}
