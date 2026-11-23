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
- `/invite` and `/kick` for private-room moderation
- A 20-character message length limit
- Per-user message rate limiting with escalating cooldowns
- Per-user room-join limiting with temporary bans
- Username/room-name validation and message sanitization against terminal injection

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
ssh -p 2222 yourname@localhost
```

Pick a name explicitly: by default `ssh` sends your local account name, which
is rejected if it contains spaces or other disallowed characters (see
[Names and input sanitization](#names-and-input-sanitization)).

The server generates `cmd/nite-chat/.ssh/host_ed25519` on first start and
reuses it on subsequent starts. Type `/listrooms` to see active rooms, then
join or create one:

```text
/listrooms
/join lounge
```

Once connected to a room, type a message and press Enter. Messages longer
than 20 characters are rejected. Messages are limited to five messages in
five seconds per user. Exceeding that limit starts a five-second cooldown;
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
| `/invite <username>` | Invite a connected user to the current private room (owner only). |
| `/kick <username>` | Disconnect a user from the current private room (owner only). |
| `/quit` | Close the chat session. |

The displayed username combines the SSH username with the last four
characters of the public-key fingerprint.

## Private rooms

When creating a room, answer `y` when asked whether it should be private.
Private rooms can only be joined by the owner and users invited with:

```text
/invite <username>
```

The private-room owner can remove a connected member with:

```text
/kick <username>
```

Ownership is tied to the owner's SSH public-key fingerprint, not their
displayed username. This allows the owner to reconnect and retain ownership.

If the owner disconnects, the private room remains available to its other
connected users. Any user who disconnects is removed from the room. The room
is automatically deleted once it becomes empty.

## Names and input sanitization

Everything one user sends ends up on other users' terminals, so input is
checked before it is shown to anyone. Otherwise a crafted name or message
could inject ANSI escape sequences (clear screens, recolour text, fake system
messages) or use invisible Unicode to spoof what others see.

- **Usernames** (the SSH login name) are lowercased and must match
  `^[a-z0-9][a-z0-9_.-]{0,23}$`. `system`, `server`, `admin` and `root` are
  reserved. Names are ASCII-only, so look-alike letters (e.g. Cyrillic `а`)
  can't be used to impersonate someone. An invalid name gets a message
  explaining the rules, then the session is closed.
- **Room names** follow the same rules (up to 32 characters) and are
  lowercased, so `General` and `general` are the same room.
- **Message bodies** keep normal Unicode text and emoji. Control characters
  (ESC, the single-byte CSI `U+009B`, DEL, …) and invisible formatting
  characters (bidi overrides like `U+202E`, zero-width spaces) are stripped.
  Messages are sanitized before the empty and length checks.
- **System messages** are shown with a `*** ` prefix, which no username can
  produce, so users can't fake server announcements.

The rules live in `internal/chat/message/validate.go`, with tests in
`validate_test.go`.

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

When any user, including the owner of a private room, disconnects, they are
removed from their current room. Private rooms remain available while they
still have connected users and are removed when empty.

## Development

Format the code:

```bash
gofmt -w ./cmd ./internal
```

Build the application:

```bash
go build ./...
```

## Local Docker testing

Build the image:

```bash
docker build -t nite-chat .
```

Create a named volume so the server's SSH host key survives container
restarts:

```bash
docker volume create nite-chat-hostkey
```

Start the container:

```bash
docker run --rm \
  --name nite-chat \
  -p 2222:2222 \
  -v nite-chat-hostkey:/app/.ssh \
  nite-chat
```

Connect from another terminal:

```bash
ssh -p 2222 localhost
```

Stop the server with `Ctrl+C`. The named volume preserves the generated host
key for the next run.
