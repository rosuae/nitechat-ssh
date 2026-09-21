# NiteChat SSH

NiteChat is a small SSH chat server written in Go.

## Running

Start the server with:

```bash
go run server.go
```

Connect from another terminal:

```bash
ssh -p 2222 localhost
```

## Architecture

The server accepts SSH connections on port `2222` and creates a session for each client.
The `Hub` tracks connected clients and broadcasts messages to them.
Each `Client` has its own buffered channel and `Listen` goroutine for terminal output.
The server's Ed25519 host key is stored in `.ssh/host_ed25519` and reused between restarts.
Type `/leave` to disconnect from the chat.
