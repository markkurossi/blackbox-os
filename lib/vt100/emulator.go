//
// Copyright (c) 2018-2021 Markku Rossi
//
// All rights reserved.
//

package vt100

import (
	"fmt"
	"io"
)

type Point struct {
	X int
	Y int
}

func (p Point) String() string {
	return fmt.Sprintf("%d,%d", p.X, p.Y)
}

func (p Point) Equal(o Point) bool {
	return p.X == o.X && p.Y == o.Y
}

var (
	ZeroPoint = Point{}
)

type RGBA uint32

const (
	White = RGBA(0xffffffff)
	Black = RGBA(0x000000ff)
	debug = false
)

type Char struct {
	Code       rune
	Foreground RGBA
	Background RGBA
}

type CharDisplay interface {
	Size() Point
	Clear(from, to Point)
	// DECALN fills the screen with 'E'.
	DECALN(size Point)
	Set(p Point, char Char)
	Get(p Point) Char
	ScrollUp(count int)
}

type Emulator struct {
	display  CharDisplay
	Size     Point
	Cursor   Point
	blank    Char
	overflow bool
	state    *State
	stdout   io.Writer
	stderr   io.Writer
}

func NewEmulator(stdout, stderr io.Writer, display CharDisplay) *Emulator {
	e := &Emulator{
		display: display,
		state:   stStart,
		stdout:  stdout,
		stderr:  stderr,
	}
	e.Reset()
	return e
}

func (e *Emulator) Reset() {
	e.Size = e.display.Size()
	e.blank = Char{
		Code:       ' ',
		Foreground: Black,
		Background: White,
	}
	e.Clear(true, true)
}

func (e *Emulator) Resize(width, height int) {
	e.Size = e.display.Size()
	if e.Size.X > width {
		e.Size.X = width
	}
	if e.Size.Y > height {
		e.Size.Y = height
	}
}

func (e *Emulator) setState(state *State) {
	e.state = state
	e.state.Reset()
}

func (e *Emulator) output(format string, a ...interface{}) {
	if e.stdout == nil {
		return
	}
	e.stdout.Write([]byte(fmt.Sprintf(format, a...)))
}

func (e *Emulator) debug(format string, a ...interface{}) {
	if e.stderr == nil {
		return
	}
	e.stderr.Write([]byte(fmt.Sprintf(format, a...)))
}

func (e *Emulator) setIconName(name string) {
	e.debug("Icon Name: %s", name)
}

func (e *Emulator) setWindowTitle(name string) {
	e.debug("Window Title: %s", name)
}

func (e *Emulator) ClearLine(line, from, to int) {
	if line < 0 || line >= e.Size.Y {
		return
	}
	if to >= e.Size.X {
		to = e.Size.X - 1
	}
	e.display.Clear(Point{
		X: from,
		Y: line,
	}, Point{
		X: to,
		Y: line,
	})
}

func (e *Emulator) Clear(start, end bool) {
	if start {
		if e.Cursor.Y > 0 {
			e.display.Clear(ZeroPoint, Point{
				X: e.Size.X - 1,
				Y: e.Cursor.Y - 1,
			})
		}
		e.display.Clear(Point{
			X: 0,
			Y: e.Cursor.Y,
		}, Point{
			X: e.Cursor.X,
			Y: e.Cursor.Y,
		})
	}
	if end {
		e.display.Clear(Point{
			X: e.Cursor.X,
			Y: e.Cursor.Y,
		}, Point{
			X: e.Size.X - 1,
			Y: e.Cursor.Y,
		})
		e.display.Clear(Point{
			Y: e.Cursor.Y + 1,
		}, Point{
			X: e.Size.X - 1,
			Y: e.Size.Y - 1,
		})
	}
}

func (e *Emulator) MoveTo(row, col int) {
	if col < 0 {
		col = 0
	}
	if col >= e.Size.X {
		col = e.Size.X - 1
	}
	e.Cursor.X = col

	if row < 0 {
		row = 0
	}
	if row >= e.Size.Y {
		e.ScrollUp(e.Size.Y - row + 1)
		row = e.Size.Y - 1
	}
	e.Cursor.Y = row
	e.overflow = false
}

func (e *Emulator) ScrollUp(count int) {
	if count >= e.Size.Y {
		e.Clear(true, true)
		return
	}
	e.display.ScrollUp(count)

	for i := 0; i < count; i++ {
		e.ClearLine(e.Size.Y-1-i, 0, e.Size.X)
	}
}

func (e *Emulator) InsertChar(code int) {
	if e.overflow {
		if e.Cursor.Y+1 >= e.Size.Y {
			e.ScrollUp(1)
			e.MoveTo(e.Cursor.Y, 0)
		} else {
			e.MoveTo(e.Cursor.Y+1, 0)
		}
		e.overflow = true
	}
	ch := e.blank
	ch.Code = rune(code)
	e.display.Set(e.Cursor, ch)
	if e.Cursor.X+1 >= e.Size.X {
		e.overflow = true
	} else {
		e.MoveTo(e.Cursor.Y, e.Cursor.X+1)
	}
}

func (e *Emulator) InsertChars(row, col, count int) {
	if row < 0 {
		row = 0
	} else if row >= e.Size.Y {
		row = e.Size.Y - 1
	}
	if col < 0 {
		col = 0
	} else if col >= e.Size.X {
		return
	}
	if col+count >= e.Size.X {
		e.ClearLine(row, col, e.Size.X)
		return
	}
	p := Point{
		Y: row,
	}
	for p.X = e.Size.X - 1; p.X >= col; p.X-- {
		if p.X-count >= col {
			e.display.Set(p, e.display.Get(Point{
				Y: row,
				X: p.X - count,
			}))
		} else {
			e.display.Set(p, e.blank)
		}
	}
}

func (e *Emulator) DeleteChars(row, col, count int) {
	p := Point{
		Y: row,
	}

	for p.X = col; p.X < e.Size.X; p.X++ {
		if p.X+count < e.Size.X {
			e.display.Set(p, e.display.Get(Point{
				Y: row,
				X: p.X + count,
			}))
		} else {
			e.display.Set(p, e.blank)
		}
	}
}

func (e *Emulator) Input(code int) {
	if debug {
		e.debug("Emulator.Input: %s<-0x%x (%d) '%c'", e.state, code, code, code)
	}
	next := e.state.Input(e, code)
	if next != nil {
		if debug {
			e.debug("Emulator.Input: %s->%s", e.state, next)
		}
		e.setState(next)
	}
}
