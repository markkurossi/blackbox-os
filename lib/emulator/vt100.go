//
// vt100.go
//
// Copyright (c) 2018 Markku Rossi
//
// All rights reserved.
//

package emulator

import (
	"io"
)

func VT100CursorForward(out io.Writer) error {
	_, err := out.Write([]byte{0x1b, '[', 'C'})
	return err
}

func VT100Backspace(out io.Writer) error {
	_, err := out.Write([]byte{0x08})
	return err
}

func VT100DeleteChar(out io.Writer) error {
	_, err := out.Write([]byte{0x1b, '[', 'P'})
	return err
}

func VT100EraseLineHead(out io.Writer) error {
	_, err := out.Write([]byte{0x1b, '[', '1', 'K'})
	return err
}

func VT100EraseLineTail(out io.Writer) error {
	_, err := out.Write([]byte{0x1b, '[', 'K'})
	return err
}

func VT100EraseLine(out io.Writer) error {
	_, err := out.Write([]byte{0x1b, '[', '2', 'K'})
	return err
}
