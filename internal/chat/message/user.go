package message

import (
	"fmt"
	"strings"
	"sync"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

// to do: notifications channel
type User struct {
	hubMan     HubManager
	currentHub Hub

	ui          *term.Terminal
	channel     ssh.Channel
	fingerprint string
	username    string
	receive     chan *Message

	closeOnce   sync.Once
	receiveOnce sync.Once
	mu          sync.Mutex
}

type HubManager interface {
	GetOrCreate(string) Hub
	ListRooms() []HubInfo
}

type Hub interface {
	RegisterUser(*User)
	UnregisterUser(*User)
	SendMessage(*Message)
	HubDetails() (string, int)
}

type HubInfo struct {
	Name  string
	Count int
}

func NewUser(hMan HubManager, fp string, username string, channel ssh.Channel) *User {
	return &User{
		hubMan:     hMan,
		currentHub: nil,

		ui:          term.NewTerminal(channel, "> "),
		channel:     channel,
		fingerprint: fp,
		username:    username,
		receive:     make(chan *Message, 32),
	}
}

func (u *User) Listen() {
	for msg := range u.receive {
		if msg == nil {
			continue
		}
		if _, err := u.ui.Write([]byte(msg.Render())); err != nil {
			fmt.Println("Failed to write to user:", err)
			return
		}
	}
}

func (u *User) Send(msg *Message) bool {
	select {
	case u.receive <- msg:
		return true
	default:
		return false
	}
}

func (u *User) CloseConn() {
	u.closeOnce.Do(func() {
		if err := u.channel.Close(); err != nil {
			fmt.Println("Failed to close client channel:", err)
		}
	})
}

func (u *User) CloseReceive() {
	u.receiveOnce.Do(func() {
		close(u.receive)
	})
}

func (u *User) Username() string {
	return u.username + "@" + u.fingerprint[len(u.fingerprint)-4:]
}

func (u *User) writeLine(s string) error {
	if _, err := u.ui.Write([]byte(s + "\r\n")); err != nil {
		fmt.Println("Failed to write to user:", err)
		return err
	}
	return nil
}

func (u *User) inputLoop() {
	for {
		line, err := u.ui.ReadLine()
		if err != nil {
			fmt.Println("User disconnected or error:", err)
			return
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "/") {
			u.handleCommand(line)
			continue
		}
		hub := u.currentHubSafe()
		if hub == nil {
			err := u.writeLine("You are not connected to any room. Use /join <room>.")
			if err != nil {
				return
			}
			continue
		}
		hub.SendMessage(NewMessage(u, line))
	}
}

func (u *User) handleCommand(line string) {
	parts := strings.Fields(line)
	cmd, args := parts[0], parts[1:]

	switch cmd {
	case "/disconnect":
		u.LeaveHub()
	case "/listrooms":
		u.listHubs()
	case "/join":
		if len(args) != 1 {
			_ = u.writeLine("Usage: /join <room>")
			return
		}
		err := u.JoinHub(args[0])
		if err != nil {
			return
		}
	default:
		_ = u.writeLine("Unknown command: " + cmd)
	}
}

func (u *User) listHubs() {
	rooms := u.hubMan.ListRooms()
	if len(rooms) == 0 {
		_ = u.writeLine("No active rooms.")
		return
	}
	var sb strings.Builder
	sb.WriteString("Active rooms:\r\n")
	for _, r := range rooms {
		fmt.Fprintf(&sb, "  %s (%d users)\r\n", r.Name, r.Count)
	}
	_ = u.writeLine(sb.String())
}

func (u *User) currentHubSafe() Hub {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.currentHub
}

func (u *User) HandleSession(requests <-chan *ssh.Request) {
	defer func() {
		if hub := u.currentHubSafe(); hub != nil {
			hub.UnregisterUser(u)
			u.mu.Lock()
			u.currentHub = nil
			u.mu.Unlock()
		}
		if err := u.channel.Close(); err != nil {
			fmt.Println("Failed to close user channel:", err)
		}
	}()

	go func() {
		for req := range requests {
			if req.Type == "pty-req" || req.Type == "shell" {
				err := req.Reply(true, nil)
				if err != nil {
					return
				}
			} else {
				err := req.Reply(false, nil)
				if err != nil {
					return
				}
			}
		}
	}()

	if err := u.writeLine("Welcome to nite-chat!\r\n Available commands: /listrooms, /join <room>, /disconnect. "); err != nil {
		return
	}

	go u.Listen()
	u.inputLoop()
}

func (u *User) JoinHub(room string) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	if u.currentHub != nil {
		oldName, _ := u.currentHub.HubDetails()
		if oldName == room {
			return nil
		}
	}

	newHub := u.hubMan.GetOrCreate(room)
	newHub.RegisterUser(u)

	if u.currentHub != nil {
		oldHub := u.currentHub
		oldName, _ := oldHub.HubDetails()
		oldHub.UnregisterUser(u)
		if err := u.writeLine("Disconnecting from " + oldName + "..."); err != nil {
			return err
		}
	}

	u.currentHub = newHub
	if err := u.writeLine("Joined room " + room + "."); err != nil {
		return err
	}
	return nil
}

func (u *User) LeaveHub() {
	u.mu.Lock()
	defer u.mu.Unlock()

	if u.currentHub == nil {
		_ = u.writeLine("You are not connected to any room. Use /join <room>.")
		return
	}

	oldName, _ := u.currentHub.HubDetails()
	u.currentHub.UnregisterUser(u)
	u.currentHub = nil
	_ = u.writeLine("Disconnecting from " + oldName + "...")
}
