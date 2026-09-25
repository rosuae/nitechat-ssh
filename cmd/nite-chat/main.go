package main

import (
	"fmt"
	"os"

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

	server := chat.NewServer(":2222", signer)
	if err := server.ListenAndServe(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
