//
// Copyright (c) 2021 Markku Rossi
//
// All rights reserved.
//

package vt100

var (
	_ CharDisplay = &Display{}
)

type Display struct {
	Blank Char
	size  Point
	Lines [][]Char
}

func NewDisplay(width, height int) *Display {
	d := &Display{
		Blank: Char{
			Code:       0xa0,
			Foreground: Black,
			Background: White,
		},
		size: Point{
			X: width,
			Y: height,
		},
	}
	d.Resize(width, height)
	return d
}

func (d *Display) Resize(width, height int) {
	d.size.X = width
	d.size.Y = height

	for row := 0; row < height; row++ {
		var line []Char
		var start int
		if row < len(d.Lines) {
			line = d.Lines[row]
			start = len(line)
		} else {
			line = make([]Char, width)
			start = 0
			d.Lines = append(d.Lines, line)
		}
		for col := start; col < width; col++ {
			line[col] = d.Blank
		}
	}
}

func (d *Display) Size() Point {
	return d.size
}

func (d *Display) Set(p Point, char Char) {
	d.Lines[p.Y][p.X] = char
}

func (d *Display) Get(p Point) Char {
	return d.Lines[p.Y][p.X]
}

func (d *Display) ScrollUp(count int) {
	for i := 0; i < count; i++ {
		saved := d.Lines[0]
		d.Lines = append(d.Lines[1:], saved)
	}
}
