package message

import (
	"fmt"
	"time"
)

type Message struct {
	sender    string
	body      string
	timestamp time.Time
}

func NewMessage(sender *User, body string) *Message {
	return &Message{
		sender:    sender.Username(),
		body:      body,
		timestamp: time.Now(),
	}
}

func (m *Message) Render() string {
	return fmt.Sprintf("%s: %s\r\n", m.sender, m.body)
}
