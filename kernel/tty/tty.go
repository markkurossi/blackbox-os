//
// tty.go
//
// Copyright (c) 2018-2021 Markku Rossi
//
// All rights reserved.
//

package tty

import (
	"github.com/markkurossi/vt100"
)

type TTYFlags uint

const (
	ICANON TTYFlags = 1 << iota
	ECHO
)

type TTY interface {
	Flags() TTYFlags
	SetFlags(flags TTYFlags)
	Read(p []byte) (n int, err error)
	Cursor() vt100.Point
	Size() (ch, px vt100.Point)
	Write(p []byte) (n int, err error)
	Flush() error
}
