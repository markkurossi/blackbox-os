//
// console.go
//
// Copyright (c) 2018 Markku Rossi
//
// All rights reserved.
//

package tty

import (
	"encoding/hex"
	"fmt"
	"log"
	"sync"
	"syscall/js"
	"unicode"

	"github.com/markkurossi/blackbox-os/kernel/control"
	"github.com/markkurossi/blackbox-os/kernel/kmsg"
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
	flags     emulator.TTYFlags
	qCanon    *Canonical
	qNonCanon []byte
	cond      *sync.Cond
	emulator  *emulator.Emulator
}

// Canonical provides canonical input mode with Emacs-like line
// editing capabilities.
type Canonical struct {
	buf    []rune
	cursor int
	tail   int
	avail  []byte
}

func (in *Canonical) input(c *Console, kt KeyType, code rune) bool {

	switch kt {
	case KeyCode:
		kmsg.Print(fmt.Sprintf("input(KeyCode, 0x%x)\n", code))
		switch code {
		case 0x01: // C-a
			for in.cursor > 0 {
				c.Echo([]int{0x08})
				in.cursor--
			}

		case 0x02: // C-b
			in.cursorLeft(c)

		case 0x04: // C-d
			if in.cursor == in.tail {
				break
			}
			c.Echo([]int{0x1b, '[', 'P'})
			in.cursor++
			in.delete()

		case 0x05: // C-e
			for in.cursor < in.tail {
				c.Echo([]int{0x1b, '[', 'C'})
				in.cursor++
			}

		case 0x06: // C-f
			in.cursorRight(c)

		case 0x0b: // C-k
			in.tail = in.cursor
			c.Echo([]int{0x1b, '[', 'K'})

		case 0x7f: // Delete
			if in.cursor == 0 {
				break
			}

			c.Echo([]int{0x08}) // Backspace
			if in.cursor == in.tail {
				c.Echo([]int{0x1b, '[', 'K'}) // Erase line from cursor
			} else {
				c.Echo([]int{0x1b, '[', 'P'}) // Delete character
			}

			in.delete()

		default:
			if code == '\n' {
				in.avail = append(in.avail, []byte(string(in.buf[:in.tail]))...)
				in.avail = append(in.avail, '\n')
				in.cursor = 0
				in.tail = 0
				return true
			}
			if unicode.IsPrint(rune(code)) {
				if in.insert(code) {
					// Print line.
					for i := in.cursor - 1; i < in.tail; i++ {
						c.Echo([]int{int(in.buf[i])})
					}
					// And move cursor back to its position.
					for i := in.tail; i > in.cursor; i-- {
						c.Echo([]int{0x08})
					}
				}
			} else {
				kmsg.Print(fmt.Sprintf("Skipping non-printable 0x%x\n", code))
			}
		}

	case KeyCursorLeft:
		in.cursorLeft(c)

	case KeyCursorRight:
		in.cursorRight(c)
	}
	return false
}

func (in *Canonical) cursorLeft(c *Console) {
	if in.cursor > 0 {
		c.Echo([]int{0x08})
		in.cursor--
	}
}

func (in *Canonical) cursorRight(c *Console) {
	if in.cursor < in.tail {
		c.Echo([]int{0x1b, '[', 'C'})
		in.cursor++
	}
}

func (in *Canonical) insert(ch rune) bool {
	if in.tail >= len(in.buf) {
		return false
	}

	if in.cursor < in.tail {
		for i := in.tail + 1; i > in.cursor; i-- {
			in.buf[i] = in.buf[i-1]
		}
	}
	in.buf[in.cursor] = ch

	in.cursor++
	in.tail++

	return true
}

func (in *Canonical) delete() {
	if in.cursor == in.tail {
		in.cursor--
		in.tail--
	} else {
		in.cursor--
		in.buf = append(in.buf[0:in.cursor], in.buf[in.cursor+1:]...)
	}
}

func NewCanonical() *Canonical {
	return &Canonical{
		buf: make([]rune, 1024),
	}
}

func (c *Console) Flags() emulator.TTYFlags {
	return c.flags
}

func (c *Console) SetFlags(flags emulator.TTYFlags) {
	c.flags = flags
}

func (c *Console) Cursor() (int, int) {
	return c.emulator.Row, c.emulator.Col
}

func (c *Console) Size() (int, int, int, int) {
	return c.emulator.Width, c.emulator.Height,
		c.emulator.Width, c.emulator.Height
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

			if j == c.emulator.Col && i == c.emulator.Row {
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
	c.cond.L.Lock()

	var n int

	if (c.flags & emulator.ICANON) != 0 {
		for len(c.qCanon.avail) == 0 {
			c.cond.Wait()
		}
		n = copy(p, c.qCanon.avail)
		c.qCanon.avail = c.qCanon.avail[n:]
	} else {
		for len(c.qNonCanon) == 0 {
			c.cond.Wait()
		}
		n = copy(p, c.qNonCanon)
		c.qNonCanon = c.qNonCanon[n:]
	}

	c.cond.L.Unlock()

	return n, nil
}

// Write implements the io.Writer interface.
func (c *Console) Write(p []byte) (int, error) {
	kmsg.Print(fmt.Sprintf("Console.Write:\n%s", hex.Dump(p)))
	var last byte
	for _, b := range p {
		if b == '\n' && last != '\r' {
			c.emulator.Input('\r')
		}
		c.emulator.Input(int(b))
		last = b
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

	if (c.flags & emulator.ICANON) != 0 {
		if c.qCanon.input(c, kt, code) {
			c.emulator.MoveTo(c.emulator.Row+1, 0)
			c.cond.Broadcast()
		}
	} else {
		c.qNonCanon = append(c.qNonCanon, []byte(string(code))...)
		c.cond.Broadcast()
	}

	c.cond.L.Unlock()
}

func (c *Console) Echo(code []int) {
	if (c.flags & emulator.ECHO) != 0 {
		for _, co := range code {
			c.emulator.Input(co)
		}
		c.Flush()
	}
}

func NewConsole() emulator.TTY {
	c := &Console{
		flags:    emulator.ICANON | emulator.ECHO,
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
