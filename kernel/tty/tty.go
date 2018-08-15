//
// tty.go
//
// Copyright (c) 2018 Markku Rossi
//
// All rights reserved.
//

package tty

type TTYFlags uint

const (
	ICANON TTYFlags = 1 << iota
	ECHO
)

type TTY interface {
	SetFlags(flags TTYFlags)
	Read(p []byte) (n int, err error)
	Write(p []byte) (n int, err error)
	Flush() error
}
