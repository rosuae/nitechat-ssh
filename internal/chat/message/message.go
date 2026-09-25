package message

import (
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

func (m Message) String() string {
	return m.body
}
