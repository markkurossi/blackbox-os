//
// console.go
//
// Copyright (c) 2018 Markku Rossi
//
// All rights reserved.
//

package tty

import (
	"fmt"
	"io"
	"log"
	"sync"
	"syscall/js"

	"github.com/markkurossi/blackbox-os/kernel/control"
	"github.com/markkurossi/blackbox-os/lib/emulator"
)

var (
	initKeyboard = js.Global().Get("initKeyboard")
	getWidth     = js.Global().Get("displayWidth")
	getHeight    = js.Global().Get("displayHeight")
	clear        = js.Global().Get("displayClear")
	addLine      = js.Global().Get("displayAddLine")
	debug        = js.Global().Get("debug")
)

type KeyType int

var keyTypeNames = map[KeyType]string{
	KeyCode:        "Code",
	KeyCursorUp:    "CursorUp",
	KeyCursorDown:  "CursorDown",
	KeyCursorLeft:  "CursorLeft",
	KeyCursorRight: "CursorRight",
	KeyPageUp:      "PageUp",
	KeyPageDown:    "PageDown",
	KeyHome:        "Home",
	KeyEnd:         "End",
}

func (t KeyType) String() string {
	name, ok := keyTypeNames[t]
	if ok {
		return name
	}
	return fmt.Sprintf("{KeyType %d}", t)
}

const (
	KeyCode KeyType = iota
	KeyCursorUp
	KeyCursorDown
	KeyCursorLeft
	KeyCursorRight
	KeyPageUp
	KeyPageDown
	KeyHome
	KeyEnd
)

type Console struct {
	Flags    TTYFlags
	qCanon   *Canonical
	cond     *sync.Cond
	emulator *emulator.Emulator
}

type Canonical struct {
	buf    []rune
	avail  int
	cursor int
	tail   int
}

func (in *Canonical) input(kt KeyType, code rune) bool {
	switch kt {
	case KeyCode:
		in.append(code)
		if code == '\n' {
			in.avail = in.tail
			in.cursor = in.tail
			log.Printf("Line: %s", string(in.buf[:in.tail]))
			return true
		}
	}
	return false
}

func (in *Canonical) append(ch rune) {
	if in.tail < len(in.buf) {
		in.buf[in.tail] = ch
		in.tail++
	}
}

func NewCanonical() *Canonical {
	return &Canonical{
		buf: make([]rune, 1024),
	}
}

func (c *Console) SetFlags(flags TTYFlags) {
	c.Flags = flags
}

func (c *Console) String() string {
	return fmt.Sprintf("Console (%dx%d)", c.emulator.Width, c.emulator.Height)
}

func (c *Console) Resize() {
	c.emulator.Resize(getWidth.Invoke().Int(), getHeight.Invoke().Int())
}

func (c *Console) Flush() error {
	clear.Invoke()

	line := make([]uint32, c.emulator.Width*4)
	ta := js.TypedArrayOf(line)

	for i := 0; i < c.emulator.Height; i++ {
		for j := 0; j < c.emulator.Width; j++ {
			ch := c.emulator.Lines[i][j]
			line[j*4] = uint32(ch.Code)
			line[j*4+1] = uint32(ch.Foreground)
			line[j*4+2] = uint32(ch.Background)

			var flags = 0

			if j == c.emulator.X && i == c.emulator.Y {
				flags = 1
			}
			line[j*4+3] = uint32(flags)
		}
		addLine.Invoke(ta)
	}
	ta.Release()

	return nil
}

// Read implements the io.Reader interface.
func (c *Console) Read(p []byte) (int, error) {
	return 0, io.EOF
}

// Write implements the io.Writer interface.
func (c *Console) Write(p []byte) (int, error) {
	for _, b := range p {
		switch b {
		case '\n':
			c.emulator.MoveTo(0, c.emulator.Y+1)

		case '\r':
			c.emulator.MoveTo(0, c.emulator.Y)

		case '\t':
			x := c.emulator.X
			for (x % 8) != 0 {
				x++
			}
			c.emulator.MoveTo(x, c.emulator.Y)

		default:
			c.emulator.InsertChar(int(b))
		}
	}

	c.Flush()

	return len(p), nil
}

func (c *Console) OnKeyEvent(evType, key string, keyCode int, ctrl bool) {
	if evType != "keydown" {
		return
	}
	if false {
		log.Printf("%s: key=%s, keyCode=%d, ctrlKey=%v\n",
			evType, key, keyCode, ctrl)
	}

	runes := []rune(key)

	if len(runes) == 1 {
		var code = runes[0]
		if ctrl {
			if 0x61 <= code && code <= 0x7a {
				code -= 0x60
			} else if code == 0x5f {
				code = 0x1f
			} else if code == 0x20 {
				code = 0x00
			}
		}
		c.onKey(KeyCode, code)
	} else {
		switch key {
		case "Enter":
			c.onKey(KeyCode, rune(0x0a))
		case "Backspace":
			c.onKey(KeyCode, rune(0x7f))
		case "Tab":
			c.onKey(KeyCode, rune(0x09))
		case "Escape":
			c.onKey(KeyCode, rune(0x1b))
		case "ArrowUp":
			c.onKey(KeyCursorUp, 0)
		case "ArrowDown":
			c.onKey(KeyCursorDown, 0)
		case "ArrowLeft":
			c.onKey(KeyCursorLeft, 0)
		case "ArrowRight":
			c.onKey(KeyCursorRight, 0)
		case "PageUp":
			c.onKey(KeyPageUp, 0)
		case "PageDown":
			c.onKey(KeyPageDown, 0)
		case "Home":
			c.onKey(KeyHome, 0)
		case "End":
			c.onKey(KeyEnd, 0)
		}
	}

	if key == "F8" {
		control.Halt()
	}
}

func (c *Console) onKey(kt KeyType, code rune) {
	c.cond.L.Lock()

	if (c.Flags & ICANON) != 0 {
		if c.qCanon.input(kt, code) {
			c.cond.Signal()
		} else if (c.Flags & ECHO) != 0 {
			c.emulator.InsertChar(int(code))
			c.Flush()
		}
	} else {
		c.inputNonCanonical(kt, code)
	}

	c.cond.L.Unlock()
}

func (c *Console) inputCanonical(kt KeyType, code rune) {
	if kt == KeyCode {
		log.Printf("Key %d (%s)", code, string(code))
	} else {
		log.Printf("%s\n", kt)
	}
}
func (c *Console) inputNonCanonical(kt KeyType, code rune) {
	log.Printf("Noncanonical input")
}

func NewConsole() TTY {
	c := &Console{
		Flags:    ICANON | ECHO,
		qCanon:   NewCanonical(),
		cond:     sync.NewCond(new(sync.Mutex)),
		emulator: emulator.NewEmulator(),
	}

	flags := js.PreventDefault | js.StopPropagation
	onKeyboard := js.NewEventCallback(flags, func(event js.Value) {
		evType := event.Get("type").String()
		key := event.Get("key").String()
		keyCode := event.Get("keyCode").Int()
		ctrlKey := event.Get("ctrlKey").Bool()
		c.OnKeyEvent(evType, key, keyCode, ctrlKey)
	})

	initKeyboard.Invoke(onKeyboard)

	c.Resize()

	return c
}
