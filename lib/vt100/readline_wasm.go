//
// Copyright (c) 2021 Markku Rossi
//
// All rights reserved.
//

package vt100

import (
	"fmt"
	"io"
)

func MakeRaw(stdin io.Reader) (uint, error) {
	switch fd := stdin.(type) {
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
