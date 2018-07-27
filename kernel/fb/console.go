//
// console.go
//
// Copyright (c) 2018 Markku Rossi
//
// All rights reserved.
//

package fb

import (
	"fmt"
	"syscall/js"
)

var (
	getWidth  = js.Global().Get("displayWidth")
	getHeight = js.Global().Get("displayHeight")
	clear     = js.Global().Get("displayClear")
	addLine   = js.Global().Get("displayAddLine")
)

type RGBA uint32

const (
	White = RGBA(0xffffffff)
	Black = RGBA(0x000000ff)
)

var (
	blank = Char{
		Code:       ' ',
		Foreground: Black,
		Background: White,
	}
)

type Char struct {
	Code       int
	Foreground RGBA
	Background RGBA
}

type Console struct {
	Width  int
	Height int
	X      int
	Y      int
	Lines  [][]Char
}

func (c *Console) String() string {
	return fmt.Sprintf("Console (%dx%d)", c.Width, c.Height)
}

func (c *Console) Resize() {
	c.Width = getWidth.Invoke().Int()
	c.Height = getHeight.Invoke().Int()

	lines := make([][]Char, c.Height)
	for i := 0; i < c.Height; i++ {
		lines[i] = make([]Char, c.Width)
		for j := 0; j < c.Width; j++ {
			lines[i][j] = blank
		}
	}

	c.Lines = lines
}

func (c *Console) ClearLine(line int) {
	if line < 0 || line >= c.Height {
		return
	}
	for i := 0; i < c.Width; i++ {
		c.Lines[line][i] = blank
	}
}

func (c *Console) Clear() {
	for i := 0; i < c.Height; i++ {
		c.ClearLine(i)
	}
}

func (c *Console) Draw() {
	clear.Invoke()

	line := make([]uint32, c.Width*3)
	ta := js.TypedArrayOf(line)

	for i := 0; i < c.Height; i++ {
		for j := 0; j < c.Width; j++ {
			c := c.Lines[i][j]
			line[j*3] = uint32(c.Code)
			line[j*3+1] = uint32(c.Foreground)
			line[j*3+2] = uint32(c.Background)
		}
		addLine.Invoke(ta)
	}
	ta.Release()
}

func (c *Console) MoveTo(x, y int) {
	if x < 0 {
		x = 0
	}
	if x > c.Width {
		x = c.Width
	}
	c.X = x

	if y < 0 {
		y = 0
	}
	if y >= c.Height {
		c.ScrollUp(c.Height - y + 1)
		y = c.Height - 1
	}
	c.Y = y
}

func (c *Console) ScrollUp(count int) {
	if count >= c.Height {
		c.Clear()
		return
	}

	for i := 0; i < count; i++ {
		saved := c.Lines[0]
		c.Lines = append(c.Lines[1:], saved)
	}
	for i := 0; i < count; i++ {
		c.ClearLine(c.Height - 1 - i)
	}
}

// Write implements the io.Writer interface.
func (c *Console) Write(p []byte) (int, error) {
	for _, b := range p {
		switch b {
		case '\n':
			c.MoveTo(0, c.Y+1)

		case '\r':
			c.MoveTo(0, c.Y)

		case '\t':
			x := c.X
			for (x % 8) != 0 {
				x++
			}
			c.MoveTo(x, c.Y)

		default:
			if c.X >= c.Width {
				c.MoveTo(0, c.Y+1)
			}
			c.Lines[c.Y][c.X] = Char{
				Code:       int(b),
				Foreground: Black,
				Background: White,
			}
			c.MoveTo(c.X+1, c.Y)
		}
	}

	c.Draw()

	return len(p), nil
}

func NewConsole() *Console {
	c := new(Console)
	c.Resize()

	return c
}
