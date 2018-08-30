//
// process.go
//
// Copyright (c) 2018 Markku Rossi
//
// All rights reserved.
//

package process

import (
	"fmt"
	"io"
	"regexp"

	"github.com/markkurossi/backup/lib/crypto/zone"
	"github.com/markkurossi/backup/lib/storage"
	"github.com/markkurossi/backup/lib/tree"
	"github.com/markkurossi/blackbox-os/kernel/tty"
)

var rePathEscape = regexp.MustCompilePOSIX("(['\"\\\\])")

type Process struct {
	TTY    tty.TTY
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	FS     *FS
}

func NewProcess(t tty.TTY, z *zone.Zone) (*Process, error) {
	fs, err := NewFS(z)
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

func NewFS(z *zone.Zone) (*FS, error) {
	fs := &FS{
		Zone: z,
	}
	// Find snapshot root.
	element, err := tree.DeserializeID(z.HeadID, z)
	if err != nil {
		// Empty filesystem.
		return nil, err
	}
	el, ok := element.(*tree.Snapshot)
	if !ok {
		return nil, fmt.Errorf("Invalid filesystem root directory: %T", element)
	}
	fs.WD = append(fs.WD, WDEntry{
		ID:   el.Root,
		Name: "",
	})

	return fs, nil
}

type FS struct {
	Zone *zone.Zone
	WD   []WDEntry
}

func (fs *FS) PWD() (storage.ID, error) {
	if len(fs.WD) > 0 {
		return fs.WD[len(fs.WD)-1].ID, nil
	}

	element, err := tree.DeserializeID(fs.Zone.HeadID, fs.Zone)
	if err != nil {
		return storage.EmptyID, err
	}

	el, ok := element.(*tree.Snapshot)
	if !ok {
		return storage.EmptyID, fmt.Errorf("Invalid root directory: %T",
			element)
	}

	return el.Root, nil
}

func (fs *FS) PWDString() string {
	str := "/"

	for idx, e := range fs.WD {
		if idx > 0 {
			str += "/"
		}
		str += e.String()
	}

	return str
}

type WDEntry struct {
	ID   storage.ID
	Name string
}

func (wd WDEntry) String() string {
	return rePathEscape.ReplaceAllString(wd.Name, "\\${1}")
}
