package message

import (
	"fmt"
	"time"
)

var messageMaxLength = 20

type Message struct {
	sender    string
	body      string
	system    bool
	timestamp time.Time
}

func NewMessage(sender *User, body string) *Message {
	return &Message{
		sender:    sender.Username(),
		body:      body,
		timestamp: time.Now(),
	}
}

func NewSystemMessage(body string) *Message {
	return &Message{
		sender:    "system",
		body:      body,
		system:    true,
		timestamp: time.Now(),
	}
}

func (m *Message) Render() string {
	if m.system {
		return fmt.Sprintf("*** %s\r\n", m.body)
	}
	return fmt.Sprintf("%s: %s\r\n", m.sender, m.body)
}
