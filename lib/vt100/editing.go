//
// vt100.go
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

func CursorForward(out io.Writer) error {
	_, err := out.Write([]byte{0x1b, '[', 'C'})
	return err
}

func Backspace(out io.Writer) error {
	_, err := out.Write([]byte{0x08})
	return err
}

func DeleteChar(out io.Writer) error {
	_, err := out.Write([]byte{0x1b, '[', 'P'})
	return err
}

func EraseLineHead(out io.Writer) error {
	_, err := out.Write([]byte{0x1b, '[', '1', 'K'})
	return err
}

func EraseLineTail(out io.Writer) error {
	_, err := out.Write([]byte{0x1b, '[', 'K'})
	return err
}

func EraseLine(out io.Writer) error {
	_, err := out.Write([]byte{0x1b, '[', '2', 'K'})
	return err
}

func EraseScreenHead(out io.Writer) error {
	_, err := out.Write([]byte{0x1b, '[', '1', 'J'})
	return err
}

func EraseScreenTail(out io.Writer) error {
	_, err := out.Write([]byte{0x1b, '[', 'J'})
	return err
}

func EraseScreen(out io.Writer) error {
	_, err := out.Write([]byte{0x1b, '[', '2', 'J'})
	return err
}

func MoveTo(out io.Writer, row, col int) error {
	_, err := out.Write([]byte(fmt.Sprintf("\x1b[%d;%dH", row, col)))
	return err
}
