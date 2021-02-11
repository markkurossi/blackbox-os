//
// process.go
//
// Copyright (c) 2018-2019, 2021 Markku Rossi
//
// All rights reserved.
//

package process

import (
	"io"

	"github.com/markkurossi/backup/lib/crypto/zone"
	"github.com/markkurossi/blackbox-os/kernel/fs"
	"github.com/markkurossi/blackbox-os/lib/vt100"
)

type Process struct {
	TTY    vt100.TTY
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	FS     *fs.FS
}

func New(t vt100.TTY, z *zone.Zone) (*Process, error) {
	fs, err := fs.New(z)
	if err != nil {
		return nil, err
	}
	return &Process{
		TTY:    t,
		Stdin:  t,
		Stdout: t,
		Stderr: t,
		FS:     fs,
	}, nil
}
