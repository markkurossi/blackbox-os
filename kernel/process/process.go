//
// process.go
//
// Copyright (c) 2018 Markku Rossi
//
// All rights reserved.
//

package process

import (
	"io"

	"github.com/markkurossi/blackbox-os/kernel/tty"
)

type Process struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

func NewProcess(t tty.TTY) *Process {
	return &Process{
		Stdin:  t,
		Stdout: t,
		Stderr: t,
	}
}
