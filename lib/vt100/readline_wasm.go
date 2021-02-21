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
		err = bbos.SetFlags(int(fd.Fd()), flags & ^(1|2))
		if err != nil {
			return 0, err
		}
		return uint(flags), nil

	default:
		return 0, fmt.Errorf("unsupported fd: %T", fd)
	}
}

func MakeCooked(stdin io.Reader, flags uint) error {
	switch fd := stdin.(type) {
	case *os.File:
		return bbos.SetFlags(int(fd.Fd()), int(flags))

	default:
		return fmt.Errorf("unsupported fd: %T", fd)
	}
}
