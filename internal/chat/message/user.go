package message

import (
	"fmt"
	"strings"
	"sync"

	"golang.org/x/crypto/ssh"
)

// to do: notifications channel
type User struct {
	hubMan     HubManager
	currentHub Hub

	tty         *Terminal
	fingerprint string
	username    string
	receive     chan *Message

	closeOnce sync.Once
	connOnce  sync.Once
	mu        sync.Mutex
}

type HubManager interface {
	GetOrCreate(string) Hub
	ListRooms() []HubInfo
	GetRoomInfo(string) (HubInfo, bool)
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
	u := &User{
		hubMan:     hMan,
		currentHub: nil,

		fingerprint: fp,
		username:    username,
		receive:     make(chan *Message, 32),
	}
	u.tty = NewTerminal(channel, u.Username())
	return u
}

func (u *User) listen() {
	for msg := range u.receive {
		if msg == nil {
			continue
		}
		if _, err := u.tty.Write([]byte(msg.Render())); err != nil {
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
	u.connOnce.Do(func() {
		if err := u.tty.Close(); err != nil {
			fmt.Println("Failed to close client channel:", err)
		}
	})
}

func (u *User) Username() string {
	return u.username + "@" + u.fingerprint[len(u.fingerprint)-4:]
}

func (u *User) inputLoop() {
	for {
		raw, err := u.tty.ReadLine()
		if err != nil {
			fmt.Println("User disconnected or error:", err)
			return
		}

		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "/") {
			if quit := u.handleCommand(line); quit { // handleCommand returns true when the user quits the server
				return
			}
			continue
		}
		if len([]rune(line)) > messageMaxLength {
			_ = u.tty.WriteLine("couldn't send, length limit reached: " + fmt.Sprint(messageMaxLength))
			continue
		}
		hub := u.currentHubSafe()
		if hub == nil {
			err := u.tty.WriteLine("You are not connected to any room. Use /join <room>.")
			if err != nil {
				return
			}
			continue
		}
		u.tty.EraseEcho(raw)
		hub.SendMessage(NewMessage(u, line))
	}
}

func (u *User) currentHubSafe() Hub {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.currentHub
}

func (u *User) HandleSession(requests <-chan *ssh.Request) {
	defer u.Close()

	go u.tty.HandleRequests(requests)
	go u.listen()

	_ = u.tty.WriteLine("Welcome to nite-chat!\r\nCommands: /listrooms, /join <room>, /disconnect, /quit")
	u.inputLoop()
}

func (u *User) Close() {
	u.closeOnce.Do(func() {
		u.leaveHub(false)
		close(u.receive)
		_ = u.tty.Close()
	})
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
		if err := u.tty.WriteLine("Disconnecting from " + oldName + "..."); err != nil {
			return err
		}
	}

	u.currentHub = newHub
	if err := u.tty.WriteLine("Joined room " + room + "."); err != nil {
		return err
	}
	return nil
}

func (u *User) LeaveHub() {
	u.leaveHub(true)
}

func (u *User) leaveHub(notify bool) {
	u.mu.Lock()
	defer u.mu.Unlock()

	if u.currentHub == nil {
		if notify {
			_ = u.tty.WriteLine("You are not connected to any room. Use /join <room>.")
		}
		return
	}

	oldName, _ := u.currentHub.HubDetails()
	u.currentHub.UnregisterUser(u)
	u.currentHub = nil
	if notify {
		_ = u.tty.WriteLine("Disconnecting from " + oldName + "...")
	}
}
