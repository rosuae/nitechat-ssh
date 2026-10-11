package message

import (
	"fmt"
	"sync"
	"sync/atomic"
	"unicode/utf8"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

type ptyReq struct {
	Term                         string
	Columns, Rows, Width, Height uint32
	Modes                        string
}

type windowChange struct {
	Columns, Rows, Width, Height uint32
}

type Terminal struct {
	ch    ssh.Channel
	prompt string
	ui    *term.Terminal
	width atomic.Int32
	wmu   sync.Mutex // serializes every write to the screen
}

func (t *Terminal) buildPrompt(text string) {
	t.prompt = text + ": "
}

func NewTerminal(ch ssh.Channel, promptText string) *Terminal {
	t := &Terminal{ch: ch}
	t.buildPrompt(promptText)
	t.ui = term.NewTerminal(ch, t.prompt)
	t.width.Store(80)
	return t
}

func (t *Terminal) ReadLine() (string, error) { return t.ui.ReadLine() }

func (t *Terminal) Write(p []byte) (int, error) {
	t.wmu.Lock()
	defer t.wmu.Unlock()
	return t.ui.Write(p)
}

func (t *Terminal) WriteLine(s string) error {
	_, err := t.Write([]byte(s + "\r\n"))
	return err
}

func (t *Terminal) EraseEcho(raw string) {
	width := int(t.width.Load())
	rows := (len(t.prompt)+utf8.RuneCountInString(raw)-1)/width + 1

	seq := make([]byte, 0, rows*8+1)
	for i := 0; i < rows; i++ {
		seq = append(seq, "\x1b[1A\x1b[2K"...)
	}
	seq = append(seq, '\r')

	t.wmu.Lock()
	defer t.wmu.Unlock()
	_, _ = t.ch.Write(seq)
}

func (t *Terminal) setSize(cols, rows int) {
	if cols <= 0 {
		return
	}
	t.width.Store(int32(cols))
	_ = t.ui.SetSize(cols, rows)
}

func (t *Terminal) HandleRequests(reqs <-chan *ssh.Request) {
	for req := range reqs {
		ok := false
		switch req.Type {
		case "pty-req":
			var p ptyReq
			if ssh.Unmarshal(req.Payload, &p) == nil {
				t.setSize(int(p.Columns), int(p.Rows))
				fmt.Println("PTY requested:", p.Term, p.Columns, p.Rows, p.Width, p.Height, p.Modes)
			}
			ok = true
		case "window-change":
			var w windowChange
			if ssh.Unmarshal(req.Payload, &w) == nil {
				t.setSize(int(w.Columns), int(w.Rows))
			}
			ok = true
		case "shell":
			ok = true
		}
		if req.WantReply {
			if err := req.Reply(ok, nil); err != nil {
				return
			}
		}
	}
}

func (t *Terminal) Close() error { return t.ch.Close() }
