//
// process.go
//
// Copyright (c) 2018-2019 Markku Rossi
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
	"github.com/markkurossi/blackbox-os/lib/file"
	"github.com/markkurossi/blackbox-os/lib/vt100"
)

type Process struct {
	TTY    vt100.TTY
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	FS     *FS
}

func (p *Process) WD() (str string, id storage.ID, err error) {
	str = p.FS.WD.String()

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
	wd, err := p.ResolvePath(path)
	if err != nil {
		return err
	}
	// Check that the last element is a directory.
	element, err := tree.DeserializeID(wd[len(wd)-1].ID, p.FS.Zone)
	if err != nil {
		return err
	}
	_, ok := element.(*tree.Directory)
	if !ok {
		return fmt.Errorf("File '%s' is not a directory", path)
	}
	p.FS.WD = wd
	return nil
}

func (p *Process) ResolvePath(filename string) (Path, error) {
	var parts []string

	if len(filename) > 0 {
		parts = file.PathSplit(filename)
	}

	var path Path
	if len(parts) == 0 || len(parts[0]) > 0 {
		// Relative path starting from the current working directory.
		path = p.FS.WD.Copy()
	} else {
		// Absolute path starting from the root.
		path = p.FS.WD[:1].Copy()
	}

	for _, part := range parts {
		switch part {
		case ".", "":
			// Stay at the current directory.

		case "..":
			// Move to parent.
			if len(path) > 1 {
				path = path[0 : len(path)-1]
			}

		default:
			// Resolve child.
			entry, err := p.FS.LookupChild(path, part)
			if err != nil {
				return nil, err
			}
			path = append(path, *entry)
		}
	}

	return path, nil
}

func NewProcess(t vt100.TTY, z *zone.Zone) (*Process, error) {
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
	fs.WD = append(fs.WD, PathElement{
		ID:   el.Root,
		Name: "",
	})

	return fs, nil
}

type FS struct {
	Zone *zone.Zone
	WD   Path
}

func (fs *FS) LookupChild(path Path, name string) (*PathElement, error) {
	if len(path) == 0 {
		return nil, fmt.Errorf("No current working directory")
	}
	element, err := tree.DeserializeID(path[len(path)-1].ID, fs.Zone)
	if err != nil {
		return nil, err
	}
	el, ok := element.(*tree.Directory)
	if !ok {
		return nil, fmt.Errorf("Invalid current working directory: %T", element)
	}
	for _, e := range el.Entries {
		if name == e.Name {
			return &PathElement{
				ID:   e.Entry,
				Name: e.Name,
			}, nil
		}
	}

	return nil, fmt.Errorf("No such file or directory '%s'", name)
}

type Path []PathElement

func (p Path) String() string {
	str := "/"
	for _, e := range p {
		if len(e.Name) > 0 {
			if str[len(str)-1] != '/' {
				str += "/"
			}
			str += e.String()
		}
	}
	return str
}

func (p Path) Copy() Path {
	result := make(Path, len(p))
	copy(result, p)
	return result
}

type PathElement struct {
	ID   storage.ID
	Name string
}

func (wd PathElement) String() string {
	return file.PathEscape(wd.Name)
}
