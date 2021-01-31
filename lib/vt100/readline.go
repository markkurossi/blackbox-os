//
// readline.go
//
// Copyright (c) 2018-2019 Markku Rossi
//
// All rights reserved.
//

package vt100

import (
	"fmt"
	"unicode"
)

type TabCompletion func(line string) (expanded string, completions []string)

type Readline struct {
	Tab    TabCompletion
	tty    TTY
	buf    []byte
	cursor int
	tail   int
}

func NewReadline(tty TTY) *Readline {
	return &Readline{
		tty: tty,
		buf: make([]byte, 1024),
	}
}

func (rl *Readline) Read(prompt string) (string, error) {
	flags := rl.tty.Flags()
	rl.tty.SetFlags(flags & ^(ICANON | ECHO))
	defer rl.tty.SetFlags(flags)

	rl.cursor = 0
	rl.tail = 0
	fmt.Fprintf(rl.tty, "%s", prompt)

	var buf [1]byte
	for {
		_, err := rl.tty.Read(buf[:])
		if err != nil {
			return rl.line(), err
		}
		if rl.input(buf[0], prompt) {
			// Line read.
			return rl.line(), nil
		}
	}
}

func (rl *Readline) line() string {
	return string(rl.buf[:rl.tail])
}

func (rl *Readline) input(b byte, prompt string) bool {
	switch b {
	case 0x01: // C-a
		for rl.cursor > 0 {
			VT100Backspace(rl.tty)
			rl.cursor--
		}

	case 0x02: // C-b
		rl.cursorLeft()

	case 0x04: // C-d
		if rl.cursor < rl.tail {
			VT100DeleteChar(rl.tty)
			rl.cursor++
			rl.delete()
		}

	case 0x05: // C-e
		for rl.cursor < rl.tail {
			rl.cursorRight()
		}

	case 0x06: // C-f
		rl.cursorRight()

	case 0x09: // TAB
		if rl.Tab != nil {
			line, completions := rl.Tab(rl.line())

			// Line contains expanded line.
			for rl.cursor > 0 {
				VT100Backspace(rl.tty)
				rl.cursor--
			}
			VT100EraseLineTail(rl.tty)

			l := []byte(line)
			rl.tail = copy(rl.buf, l)
			rl.cursor = rl.tail

			rl.tty.Write(rl.buf[:rl.tail])

			// Print completions.
			if len(completions) > 0 {
				fmt.Fprintf(rl.tty, "\n")
				Tabulate(completions, rl.tty)
				fmt.Fprintf(rl.tty, "%s", prompt)
				rl.tty.Write(rl.buf[:rl.tail])
			}
		}

	case 0x0b: // C-k
		rl.tail = rl.cursor
		VT100EraseLineTail(rl.tty)

	case 0x0c: // C-l
		VT100EraseScreen(rl.tty)
		VT100MoveTo(rl.tty, 0, 0)
		fmt.Fprintf(rl.tty, "%s", prompt)
		rl.tty.Write(rl.buf[:rl.tail])

	case 0x7f: // Delete
		if rl.cursor == 0 {
			break
		}
		VT100Backspace(rl.tty)
		if rl.cursor == rl.tail {
			VT100EraseLineTail(rl.tty)
		} else {
			VT100DeleteChar(rl.tty)
		}
		rl.delete()

	default:
		if b == '\n' {
			return true
		}
		if unicode.IsPrint(rune(b)) {
			rl.insert(b)

			// Print line.
			rl.tty.Write(rl.buf[rl.cursor-1 : rl.tail])

			// Move cursor back to its position.
			for i := rl.tail; i > rl.cursor; i-- {
				VT100Backspace(rl.tty)
			}
		} else {
			fmt.Printf("Skipping non-printable 0x%x\n", b)
		}
	}
	return false
}

func (rl *Readline) cursorLeft() {
	if rl.cursor > 0 {
		VT100Backspace(rl.tty)
		rl.cursor--
	}
}

func (rl *Readline) cursorRight() {
	if rl.cursor < rl.tail {
		VT100CursorForward(rl.tty)
		rl.cursor++
	}
}

func (rl *Readline) insert(b byte) bool {
	if rl.tail >= len(rl.buf) {
		return false
	}

	if rl.cursor < rl.tail {
		for i := rl.tail + 1; i > rl.cursor; i-- {
			rl.buf[i] = rl.buf[i-1]
		}
	}
	rl.buf[rl.cursor] = b

	rl.cursor++
	rl.tail++

	return true
}

func (rl *Readline) delete() {
	if rl.cursor == rl.tail {
		rl.cursor--
	} else {
		rl.cursor--
		rl.buf = append(rl.buf[0:rl.cursor], rl.buf[rl.cursor+1:]...)
	}
	rl.tail--
}
