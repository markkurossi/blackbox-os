//
// console.go
//
// Copyright (c) 2018-2021, 2023 Markku Rossi
//
// All rights reserved.
//

package tty

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"image/color"
	"sync"
	"syscall/js"
	"unicode"
	"unicode/utf8"

	"github.com/markkurossi/blackbox-os/kernel/control"
	"github.com/markkurossi/blackbox-os/kernel/kmsg"
	"github.com/markkurossi/vt100"
)

var (
	initKeyboard = js.Global().Get("initKeyboard")
	display      = js.Global().Get("display")
	lineNew      = js.Global().Get("Line")
	debug        = js.Global().Get("debug")
)

var (
	_ TTY = &Console{}
)

type KeyType int

var keyTypeNames = map[KeyType]string{
	KeyCode:        "Code",
	KeyEnter:       "Enter",
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
	KeyEnter
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
	flags       TTYFlags
	qCanon      *Canonical
	qNonCanon   []byte
	cond        *sync.Cond
	encodingBuf []byte
	lastRune    rune
	emulator    *vt100.Emulator
	display     *vt100.Display
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

		case 0x0c: // C-l
			c.Echo([]int{0x1b, '[', 'J'})

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
				in.newline()
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
				kmsg.Printf("console: skipping non-printable 0x%x\n", code)
			}
		}

	case KeyEnter:
		in.newline()
		return true

	case KeyCursorLeft:
		in.cursorLeft(c)

	case KeyCursorRight:
		in.cursorRight(c)
	}
	return false
}

func (in *Canonical) newline() {
	in.avail = append(in.avail, []byte(string(in.buf[:in.tail]))...)
	in.avail = append(in.avail, '\n')
	in.cursor = 0
	in.tail = 0
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

func (c *Console) Flags() TTYFlags {
	return c.flags
}

func (c *Console) SetFlags(flags TTYFlags) {
	c.flags = flags
}

func (c *Console) Cursor() vt100.Point {
	return c.emulator.Cursor
}

func (c *Console) Size() (vt100.Point, vt100.Point) {
	return c.emulator.Size, c.emulator.Size
}

func (c *Console) String() string {
	return fmt.Sprintf("Console (%s)", c.emulator.Size)
}

func (c *Console) DisplaySize() (int, int) {
	return display.Get("width").Int(), display.Get("height").Int()
}

func (c *Console) Flush() error {
	display.Call("clear")

	for i := 0; i < c.emulator.Size.Y; i++ {
		line := lineNew.New()

		for j := 0; j < c.emulator.Size.X; j++ {
			ch := c.display.Lines[i][j]

			var flags = 0
			if j == c.emulator.Cursor.X && i == c.emulator.Cursor.Y {
				flags = 1
			}

			line.Call("add", int(ch.Code), nrgbaToInt(ch.Foreground),
				nrgbaToInt(ch.Background), int(flags))
		}
		line.Call("flush")
		display.Call("addLine", line)
	}

	return nil
}

func nrgbaToInt(c color.NRGBA) int {
	return int(c.R)<<24 | int(c.G)<<16 | int(c.B)<<8 | int(c.A)

}

// Read implements the io.Reader interface.
func (c *Console) Read(p []byte) (int, error) {
	c.cond.L.Lock()

	var n int

	if (c.flags & ICANON) != 0 {
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
	if false {
		kmsg.Printf("Console.Write:\n%s", hex.Dump(p))
	}

	c.encodingBuf = append(c.encodingBuf, p...)

	for utf8.FullRune(c.encodingBuf) {
		r, size := utf8.DecodeRune(c.encodingBuf)
		c.encodingBuf = c.encodingBuf[size:]
		if r == utf8.RuneError {
			break
		}
		if r == '\n' && c.lastRune != '\r' {
			c.emulator.Input('\r')
		}
		c.emulator.Input(int(r))
		c.lastRune = r
	}

	c.Flush()

	return len(p), nil
}

func (c *Console) OnKeyEvent(evType, key string, keyCode int, ctrl bool) {
	if evType != "keydown" {
		return
	}
	if false {
		kmsg.Printf("%s: key=%s, keyCode=%d, ctrlKey=%v\n",
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
			c.onKey(KeyEnter, 0)
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
	defer c.cond.L.Unlock()

	if (c.flags & ICANON) != 0 {
		if c.qCanon.input(c, kt, code) {
			c.emulator.Input('\r')
			c.emulator.Input('\n')
			c.cond.Broadcast()
		}
	} else {
		input := new(bytes.Buffer)

		switch kt {
		case KeyCode:
			input.Write([]byte(string(code)))

		case KeyEnter:
			input.Write([]byte{'\r'})

		case KeyCursorUp:
			vt100.CursorUp(input)

		case KeyCursorDown:
			vt100.CursorDown(input)

		case KeyCursorLeft:
			vt100.CursorBackward(input)

		case KeyCursorRight:
			vt100.CursorForward(input)

		case KeyPageUp:
			vt100.ScrollUp(input)

		case KeyPageDown:
			vt100.ScrollDown(input)

		case KeyHome, KeyEnd:
			kmsg.Printf("onKey: %s not supported", kt)
			return
		}
		c.qNonCanon = append(c.qNonCanon, input.Bytes()...)
		c.cond.Broadcast()
	}
}

func (c *Console) Echo(code []int) {
	if (c.flags & ECHO) != 0 {
		for _, co := range code {
			c.emulator.Input(co)
		}
		c.Flush()
	}
}

type inputWriter struct {
	c *Console
}

// Write implements the io.Writer interface.
func (iw *inputWriter) Write(p []byte) (int, error) {
	for _, r := range string(p) {
		iw.c.onKey(KeyCode, r)
	}
	return len(p), nil
}

func NewConsole() TTY {
	c := &Console{
		flags:  ICANON | ECHO,
		qCanon: NewCanonical(),
		cond:   sync.NewCond(new(sync.Mutex)),
	}
	c.display = vt100.NewDisplay(c.DisplaySize())
	c.emulator = vt100.NewEmulator(&inputWriter{
		c: c,
	}, kmsg.Writer, c.display)

	onKeyboard := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) < 1 {
			kmsg.Printf("Invalid event arguments: %v\n", args)
			return nil
		}
		event := args[0]
		evType := event.Get("type").String()
		key := event.Get("key").String()
		keyCode := event.Get("keyCode").Int()
		ctrlKey := event.Get("ctrlKey").Bool()
		c.OnKeyEvent(evType, key, keyCode, ctrlKey)

		event.Call("stopPropagation")
		event.Call("preventDefault")

		return nil
	})

	initKeyboard.Invoke(onKeyboard)

	return c
}
