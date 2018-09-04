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

	"github.com/markkurossi/backup/lib/crypto/zone"
	"github.com/markkurossi/backup/lib/storage"
	"github.com/markkurossi/backup/lib/tree"
	"github.com/markkurossi/blackbox-os/lib/emulator"
	"github.com/markkurossi/blackbox-os/lib/file"
)

type Process struct {
	TTY    emulator.TTY
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	FS     *FS
}

func (p *Process) WD() (str string, id storage.ID, err error) {
	str = "/"
	for _, e := range p.FS.WD {
		if len(e.Name) > 0 {
			if str[len(str)-1] != '/' {
				str += "/"
			}
			str += e.String()
		}
	}

	if len(p.FS.WD) > 0 {
		id = p.FS.WD[len(p.FS.WD)-1].ID
		return
	}

	var element tree.Element

	element, err = tree.DeserializeID(p.FS.Zone.HeadID, p.FS.Zone)
	if err != nil {
		return
	}

	el, ok := element.(*tree.Snapshot)
	if !ok {
		err = fmt.Errorf("Invalid root directory: %T", element)
		return
	}
	id = el.Root
	return
}

func (p *Process) SetWD(path string) error {
	if len(path) == 0 {
		return fmt.Errorf("Invalid path '%s'", path)
	}
	parts := file.PathSplit(path)

	var wd []WDEntry
	if len(parts[0]) == 0 {
		// Absolute path starting from the root.
		wd = p.FS.WD[:1]
	} else {
		// Relative path starting from the current working directory.
		wd = p.FS.WD
	}

	for _, part := range parts {
		switch part {
		case ".", "":
			// Stay at the current directory.

		case "..":
			// Move to parent.
			if len(wd) > 1 {
				wd = wd[0 : len(wd)-1]
			}

		default:
			// Resolve sub-directory.
			entry, err := p.FS.LookupDir(wd, part)
			if err != nil {
				return err
			}
			wd = append(wd, *entry)
		}
	}
	p.FS.WD = wd
	return nil
}

func NewProcess(t emulator.TTY, z *zone.Zone) (*Process, error) {
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

func (fs *FS) LookupDir(wd []WDEntry, name string) (*WDEntry, error) {
	if len(wd) == 0 {
		return nil, fmt.Errorf("No current working directory")
	}
	element, err := tree.DeserializeID(wd[len(wd)-1].ID, fs.Zone)
	if err != nil {
		return nil, err
	}
	el, ok := element.(*tree.Directory)
	if !ok {
		return nil, fmt.Errorf("Invalid current working directory: %T", element)
	}
	for _, e := range el.Entries {
		if name == e.Name {
			return &WDEntry{
				ID:   e.Entry,
				Name: e.Name,
			}, nil
		}
	}

	return nil, fmt.Errorf("No such directory '%s'", name)
}

type WDEntry struct {
	ID   storage.ID
	Name string
}

func (wd WDEntry) String() string {
	return file.PathEscape(wd.Name)
}
