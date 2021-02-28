//
// Copyright (c) 2021 Markku Rossi
//
// All rights reserved.
//

package vt100

import (
	"io"
	"math"
)

var (
	_      CharDisplay = &Stringer{}
	stdout             = io.Discard
	stderr             = io.Discard
)

type Stringer struct {
	lines [][]rune
}

func NewStringer() *Stringer {
	return &Stringer{}
}

func (d *Stringer) Size() Point {
	return Point{
		X: math.MaxInt32,
		Y: math.MaxInt32,
	}
}

func (d *Stringer) Clear(from, to Point) {
}

func (d *Stringer) DECALN(size Point) {
	if size.X == math.MaxInt32 {
		// Take the maximum line width.
		size.X = 0
		for _, line := range d.lines {
			if len(line) > size.X {
				size.X = len(line)
			}
		}
		if size.X == 0 {
			size.X = 80
		}
	}
	if size.Y == math.MaxInt32 {
		size.Y = len(d.lines)
		if size.Y == 0 {
			size.Y = 24
		}
	}
	ch := Char{
		Code: 'E',
	}
	var pt Point
	for pt.Y = 0; pt.Y < size.Y; pt.Y++ {
		for pt.X = 0; pt.X < size.X; pt.X++ {
			d.Set(pt, ch)
		}
	}
}

func (d *Stringer) Set(p Point, char Char) {
	for len(d.lines) <= p.Y {
		d.lines = append(d.lines, []rune{})
	}
	for len(d.lines[p.Y]) <= p.X {
		d.lines[p.Y] = append(d.lines[p.Y], ' ')
	}
	d.lines[p.Y][p.X] = char.Code
}

func (d *Stringer) Get(p Point) Char {
	if p.Y >= len(d.lines) || p.X >= len(d.lines[p.Y]) {
		return Char{
			Code: ' ',
		}
	}
	return Char{
		Code: d.lines[p.Y][p.X],
	}
}

func (d *Stringer) ScrollUp(count int) {
}

// DisplayWidth computes the character size width of the argument data
// when all emulator control codes have been removed.
func DisplayWidth(data string) (width, height int, err error) {
	disp := NewStringer()
	emul := NewEmulator(stdout, stderr, disp)
	for _, r := range []rune(data) {
		emul.Input(int(r))
	}

	for _, line := range disp.lines {
		if len(line) > width {
			width = len(line)
		}
	}
	height = len(disp.lines)

	return
}

// Trim removes all emulator control codes from the argument data.
func Trim(data string) (lines []string, err error) {
	disp := NewStringer()
	emul := NewEmulator(stdout, stderr, disp)
	for _, r := range []rune(data) {
		emul.Input(int(r))
	}

	for _, line := range disp.lines {
		lines = append(lines, string(line))
	}

	return
}
