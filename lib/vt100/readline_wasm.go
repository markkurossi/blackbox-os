//
// Copyright (c) 2021 Markku Rossi
//
// All rights reserved.
//

package vt100

import (
	"fmt"
	"io"
	"os"

	"github.com/markkurossi/blackbox-os/lib/bbos"
)

func MakeRaw(stdin io.Reader) (uint, error) {
	switch fd := stdin.(type) {
	case *os.File:
		flags, err := bbos.GetFlags(int(fd.Fd()))
		if err != nil {
			return 0, err
		}
		// XXX set flags
		return uint(flags), nil

	default:
		tty, ok := stdin.(TTY)
		if ok {
			flags := tty.Flags()
			tty.SetFlags(flags & ^(ICANON | ECHO))
			return uint(flags), nil
		}
		return 0, fmt.Errorf("unsupported fd: %T", fd)
	}
}

func MakeCooked(stdin io.Reader, flags uint) error {
	switch fd := stdin.(type) {
	default:
		tty, ok := stdin.(TTY)
		if ok {
			tty.SetFlags(TTYFlags(flags))
			return nil
		}
		return fmt.Errorf("unsupported fd: %T", fd)
	}
}
