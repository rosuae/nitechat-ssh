# NiteChat SSH

NiteChat is a lightweight terminal chat server written in Go. Users connect
over SSH, join the shared chat, and exchange messages without needing a
separate client application.

## Features

- SSH public-key authentication
- Optional public-key allowlist
- One chat session per connected user
- Hub-based message broadcasting
- Persistent Ed25519 server host key
- `/leave` command for cleanly disconnecting

## Requirements

- Go 1.27 or newer
- An SSH client

## Quick start

Start the server:

```bash
go run ./cmd/nite-chat
```

The server listens on port `2222` by default. Connect from another terminal:

```bash
ssh -p 2222 localhost
```

After connecting, type a message and press Enter. Leave the chat with:

```text
/leave
```

## Connecting with a key

If the server is configured to accept only authorized public keys, connect
with the corresponding private key:

```bash
ssh -i ~/.ssh/id_ed25519 -p 2222 localhost
```

The server identifies users by their SSH username and the last four characters
of their public-key fingerprint.

## Architecture

The executable entry point is in `cmd/nite-chat`. The SSH server and chat
session lifecycle are implemented in `internal/chat`, while host-key loading
is handled by `internal/hostkey`.

Each SSH connection creates a `message.User`. The user reads input from its
session and sends messages to the chat `Hub`. The currentHub tracks registered users
and broadcasts each message to all connected users, including its sender.
Each user has a receive channel and a listener goroutine that writes incoming
messages to that user's SSH terminal.

```text
SSH client
    |
    v
message.User ---> Hub ---> connected users
      |                       |
      +--- input              +--- terminal output
```

## Project layout

```text
cmd/nite-chat/        Application entry point
internal/chat/        SSH server, currentHub, and session management
internal/chat/message/
                      Users and chat messages
internal/hostkey/     Ed25519 host-key loading
```

The server's Ed25519 host key is stored in `.ssh/host_ed25519` and reused
between restarts.

## Development

Run the application:

```bash
go run ./cmd/nite-chat
```

Format the code:

```bash
gofmt -w .
```

Build the application:

```bash
go build ./...
```
