# NiteChat SSH

NiteChat is a lightweight multi-room terminal chat server written in Go.
Users connect over SSH, choose a room, and exchange short messages without
needing a separate client application.

## Features

- SSH public-key authentication with a persistent Ed25519 host key
- Multiple in-memory chat rooms with room membership and user counts
- Room creation on demand, with empty rooms removed automatically
- Broadcast messages to everyone in the current room
- Basic SSH terminal and PTY support
- `/disconnect` and `/quit` commands for leaving a room or session
- A 20-character message length limit
- Per-user message rate limiting with escalating cooldowns
- Per-user room-join limiting with temporary bans

## Requirements

- Go 1.27.1 or newer
- An SSH client

## Quick start

Start the server from the repository root:

```bash
go run ./cmd/nite-chat
```

The server listens on TCP port `2222` by default. Connect from another
terminal:

```bash
ssh -p 2222 localhost
```

The server generates `cmd/nite-chat/.ssh/host_ed25519` on first start and
reuses it on subsequent starts. Type `/listrooms` to see active rooms, then
join or create one:

```text
/listrooms
/join lounge
```

Once connected to a room, type a message and press Enter. Messages longer
than 20 characters are rejected. Messages are limited to five messages in
five seconds per user. Exceeding that limit starts a one-second cooldown;
repeated violations double the cooldown up to 30 seconds. Ten room-join
attempts within one minute result in a 30-second temporary ban from joining
rooms.

## Client commands

Available after connecting:

| Command | Description |
| --- | --- |
| `/listrooms` | List active rooms and their user counts. |
| `/join <room>` | Join an existing room or confirm creation of a new room. |
| `/disconnect` | Leave the current room without closing the SSH session. |
| `/quit` | Close the chat session. |

The displayed username combines the SSH username with the last four
characters of the public-key fingerprint.

## Stopping the server

In the terminal running the server, enter:

```text
/quit
```

This shuts down the TCP listener cleanly. `/quit` entered in an SSH session
only closes that client session.

## Architecture

The executable entry point is in `cmd/nite-chat`. The SSH server and chat
session lifecycle are implemented in `internal/chat`, while host-key
generation and loading are handled by `internal/hostkey`.

Each SSH connection creates a `message.User`. The user reads input from its
terminal and sends messages to the selected `Hub`. A `HubManager` creates,
lists, and removes rooms. Each hub tracks its registered users and broadcasts
messages to all connected users, including the sender. Each user has a
buffered receive channel and a listener goroutine that writes incoming
messages to the SSH terminal.

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
cmd/nite-chat/             Application entry point and server console
internal/chat/             SSH server, hubs, and session management
internal/chat/message/     Users, commands, terminals, and chat messages
internal/hostkey/          Ed25519 host-key generation and loading
```

Rooms and messages are held in memory and are not persisted between server
restarts. The server's Ed25519 host key is stored in
`cmd/nite-chat/.ssh/host_ed25519` and reused between restarts.

## Development

Format the code:

```bash
gofmt -w ./cmd ./internal
```

Build the application:

```bash
go build ./...
```
