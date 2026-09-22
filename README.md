# NiteChat SSH

NiteChat is a small SSH chat server written in Go.

## Running

Start the server with:

```bash
go run ./cmd/nite-chat
```

Connect from another terminal:

```bash
ssh -p 2222 localhost
```

## Architecture

The executable is in `cmd/nite-chat`. Chat sessions and the SSH server are in
`internal/chat`, while Ed25519 host-key loading is in `internal/hostkey`.
The server accepts SSH connections on port `2222` and creates a session for each client.
The `Hub` tracks connected clients and broadcasts messages to them.
Each `Client` has its own channel and listener goroutine for terminal output.
The server's Ed25519 host key is stored in `.ssh/host_ed25519` and reused between restarts.
Type `/leave` to disconnect from the chat.
