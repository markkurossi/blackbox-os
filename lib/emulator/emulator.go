//
// emulator.go
//
// Copyright (c) 2018 Markku Rossi
//
// All rights reserved.
//

package emulator

type RGBA uint32

const (
	White = RGBA(0xffffffff)
	Black = RGBA(0x000000ff)
)

var (
	blank = Char{
		Code:       0xa0,
		Foreground: Black,
		Background: White,
	}
)

type Char struct {
	Code       int
	Foreground RGBA
	Background RGBA
}

type Emulator struct {
	Width  int
	Height int
	Col    int
	Row    int
	Lines  [][]Char
}

func (e *Emulator) Resize(width, height int) {
	e.Width = width
	e.Height = height

	lines := make([][]Char, e.Height)
	for i := 0; i < e.Height; i++ {
		lines[i] = make([]Char, e.Width)
		for j := 0; j < e.Width; j++ {
			lines[i][j] = blank
		}
	}

	e.Lines = lines
}

func (e *Emulator) ClearLine(line int) {
	if line < 0 || line >= e.Height {
		return
	}
	for i := 0; i < e.Width; i++ {
		e.Lines[line][i] = blank
	}
}

func (e *Emulator) Clear() {
	for i := 0; i < e.Height; i++ {
		e.ClearLine(i)
	}
}

func (e *Emulator) MoveTo(row, col int) {
	if col < 0 {
		col = 0
	}
	if col > e.Width {
		col = e.Width
	}
	e.Col = col

	if row < 0 {
		row = 0
	}
	if row >= e.Height {
		e.ScrollUp(e.Height - row + 1)
		row = e.Height - 1
	}
	e.Row = row
}

func (e *Emulator) ScrollUp(count int) {
	if count >= e.Height {
		e.Clear()
		return
	}

	for i := 0; i < count; i++ {
		saved := e.Lines[0]
		e.Lines = append(e.Lines[1:], saved)
	}
	for i := 0; i < count; i++ {
		e.ClearLine(e.Height - 1 - i)
	}
}

func (e *Emulator) InsertChar(code int) {
	if e.Col >= e.Width {
		e.MoveTo(e.Row+1, 0)
	}
	e.Lines[e.Row][e.Col] = Char{
		Code:       code,
		Foreground: Black,
		Background: White,
	}
	e.MoveTo(e.Row, e.Col+1)
}

func NewEmulator() *Emulator {
	return new(Emulator)
}
