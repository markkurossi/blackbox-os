//
// tty.go
//
// Copyright (c) 2018 Markku Rossi
//
// All rights reserved.
//

package tty

const (
	ICANON int = 1 << iota
	ECHO
)

type TTY interface {
	SetFlags(flags int)
	Read(p []byte) (n int, err error)
	Write(p []byte) (n int, err error)
	Flush() error
}
