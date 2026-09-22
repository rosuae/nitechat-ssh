package main

import (
	"fmt"
	"os"
	"path/filepath"

	"nite-chat/internal/chat"
	"nite-chat/internal/hostkey"
)

func main() {
	signer, err := hostkey.LoadSigner(filepath.Join(".ssh", "host_ed25519"))
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
