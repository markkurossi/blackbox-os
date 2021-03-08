//
// readline.go
//
// Copyright (c) 2018-2021 Markku Rossi
//
// All rights reserved.
//

package readline

import (
	"github.com/markkurossi/vt100"
)

import (
	"fmt"
	"io"
	"unicode"
)

// TabCompletion provides tab completions for the line.
type TabCompletion func(line string) (expanded string, completions []string)

// Mask defineshow readline outputs are masked.
type Mask int

// Output mask types.
const (
	MaskNone Mask = iota
	MaskAsterisk
)

// Readline implements interactive line reader.
type Readline struct {
	Tab    TabCompletion
	Mask   Mask
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
	buf    []byte
	state  rlState
	cursor int
	tail   int
}

type rlState func(rl *Readline, b byte, prompt string) bool

// NewReadline creates a new readline instance.
func NewReadline(stdin io.Reader, stdout, stderr io.Writer) *Readline {
	return &Readline{
		stdin:  stdin,
		stdout: stdout,
		stderr: stderr,
		buf:    make([]byte, 1024),
		state:  rlStart,
	}
}

func (rl *Readline) Read(prompt string) (string, error) {
	flags, err := MakeRaw(rl.stdin)
	if err != nil {
		return "", err
	}
	defer MakeCooked(rl.stdin, flags)

	rl.cursor = 0
	rl.tail = 0
	fmt.Fprintf(rl.stdout, "%s", prompt)

	var buf [1]byte
	for {
		_, err := rl.stdin.Read(buf[:])
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
	return rl.state(rl, b, prompt)
}

func (rl *Readline) output(b []byte) {
	switch rl.Mask {
	case MaskNone:
		rl.stdout.Write(b)

	case MaskAsterisk:
		for i := 0; i < len(b); i++ {
			rl.stdout.Write([]byte{'*'})
		}
	}
}

func rlStart(rl *Readline, b byte, prompt string) bool {
	switch b {
	case 0x1b: // ESC
		rl.state = rlESC
		return false

	case 0x01: // C-a
		for rl.cursor > 0 {
			vt100.Backspace(rl.stdout)
			rl.cursor--
		}

	case 0x02: // C-b
		rl.cursorLeft()

	case 0x04: // C-d
		if rl.cursor < rl.tail {
			vt100.DeleteChar(rl.stdout)
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
				vt100.Backspace(rl.stdout)
				rl.cursor--
			}
			vt100.EraseLineTail(rl.stdout)

			l := []byte(line)
			rl.tail = copy(rl.buf, l)
			rl.cursor = rl.tail

			rl.output(rl.buf[:rl.tail])

			// Print completions.
			if len(completions) > 0 {
				fmt.Fprintf(rl.stdout, "\n")
				Tabulate(completions, rl.stdout)
				fmt.Fprintf(rl.stdout, "%s", prompt)
				rl.output(rl.buf[:rl.tail])
			}
		}

	case 0x0b: // C-k
		rl.tail = rl.cursor
		vt100.EraseLineTail(rl.stdout)

	case 0x0c: // C-l
		vt100.EraseScreen(rl.stdout)
		vt100.MoveTo(rl.stdout, 0, 0)
		fmt.Fprintf(rl.stdout, "%s", prompt)
		rl.output(rl.buf[:rl.tail])

	case 0x7f: // Delete
		if rl.cursor == 0 {
			break
		}
		vt100.Backspace(rl.stdout)
		if rl.cursor == rl.tail {
			vt100.EraseLineTail(rl.stdout)
		} else {
			vt100.DeleteChar(rl.stdout)
		}
		rl.delete()

	default:
		if b == '\n' || b == '\r' {
			return true
		}
		if unicode.IsPrint(rune(b)) {
			rl.insert(b)

			// Print line.
			rl.output(rl.buf[rl.cursor-1 : rl.tail])

			// Move cursor back to its position.
			for i := rl.tail; i > rl.cursor; i-- {
				vt100.Backspace(rl.stdout)
			}
		} else {
			fmt.Fprintf(rl.stderr, "readline: skipping non-printable 0x%x\n", b)
		}
	}
	return false
}

func rlESC(rl *Readline, b byte, prompt string) bool {
	switch b {
	case '[':
		rl.state = rlCSI

	default:
		fmt.Fprintf(rl.stderr, "readline: ESC: unsupported: b=0x%x", b)
		rl.state = rlStart
	}
	return false
}

func rlCSI(rl *Readline, b byte, prompt string) bool {
	switch b {
	case 'C':
		rl.cursorRight()
	case 'D':
		rl.cursorLeft()
	default:
		fmt.Fprintf(rl.stderr, "readline: CSI: unsupported: b=0x%x", b)
	}
	rl.state = rlStart
	return false
}

func (rl *Readline) cursorLeft() {
	if rl.cursor > 0 {
		vt100.Backspace(rl.stdout)
		rl.cursor--
	}
}

func (rl *Readline) cursorRight() {
	if rl.cursor < rl.tail {
		vt100.CursorForward(rl.stdout)
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
