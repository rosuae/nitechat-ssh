package message

import (
	"fmt"
	"strings"
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
		err := u.JoinHub(args[0])
		if err != nil {
			return false
		}
	default:
		_ = u.tty.WriteLine("Unknown command: " + cmd)
	}
	return false
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
