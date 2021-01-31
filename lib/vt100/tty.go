//
// tty.go
//
// Copyright (c) 2018 Markku Rossi
//
// All rights reserved.
//

package vt100

type TTYFlags uint

const (
	ICANON TTYFlags = 1 << iota
	ECHO
)

type TTY interface {
	Flags() TTYFlags
	SetFlags(flags TTYFlags)
	Read(p []byte) (n int, err error)
	Cursor() (row, col int)
	Size() (width, height, widthPx, heightPx int)
	Write(p []byte) (n int, err error)
	Flush() error
}
