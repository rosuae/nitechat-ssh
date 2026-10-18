package message

import (
	"fmt"
	"strings"
	"time"
)

func (u *User) handleCommand(line string) bool {
	parts := strings.Fields(line)
	cmd, args := parts[0], parts[1:]

	switch cmd {
	case "/disconnect":
		u.LeaveHub()
	case "/quit":
		_ = u.tty.WriteLine("Thanks for coming by!")
		u.Close()
		return true
	case "/listrooms":
		u.listHubs()
	case "/join":
		if len(args) != 1 {
			_ = u.tty.WriteLine("Usage: /join <room>")
			return false
		}
		err := u.handleJoin(args[0])
		if err != nil {
			return false
		}
	default:
		_ = u.tty.WriteLine("Unknown command: " + cmd)
	}
	return false
}

func (u *User) handleJoin(hub string) error {
	if allowed, retryAfter := u.joinRateLimiter.Allow(time.Now()); !allowed {
		return u.tty.WriteLine("Too many room join attempts. Try again in " + formatRetryAfter(retryAfter) + ".")
	}

	if _, ok := u.hubMan.GetRoomInfo(hub); ok {
		return u.JoinHub(hub)
	}

	answer, err := u.interogate("That room does not exist. Create it? [y/N]")
	if err != nil {
		return err
	}

	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "y", "yes":
		return u.JoinHub(hub)
	case "", "n", "no":
		return nil
	default:
		return u.tty.WriteLine("Room not created. Please answer yes or no.")
	}
}

func (u *User) interogate(question string) (string, error) {
	if err := u.tty.WriteLine(question); err != nil {
		return "", err
	}

	answer, err := u.tty.ReadLine()
	if err != nil {
		return "", err
	}
	u.tty.EraseEcho(answer)
	return answer, nil
}

func (u *User) listHubs() {
	rooms := u.hubMan.ListRooms()
	if len(rooms) == 0 {
		_ = u.tty.WriteLine("No active rooms.")
		return
	}
	var sb strings.Builder
	sb.WriteString("Active rooms:\r\n")
	for _, r := range rooms {
		fmt.Fprintf(&sb, "  %s (%d users)\r\n", r.Name, r.Count)
	}
	_ = u.tty.WriteLine(sb.String())
}
