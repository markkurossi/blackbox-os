//
// fs.go
//
// Copyright (c) 2021 Markku Rossi
//
// All rights reserved.
//

package fs

import (
	"fmt"

	"github.com/markkurossi/backup/lib/crypto/zone"
	"github.com/markkurossi/backup/lib/storage"
	"github.com/markkurossi/backup/lib/tree"
	"github.com/markkurossi/blackbox-os/lib/file"
)

func New(z *zone.Zone) (*FS, error) {
	fs := &FS{
		zone: z,
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
	fs.wd = append(fs.wd, PathElement{
		ID:   el.Root,
		Name: "",
	})

	return fs, nil
}

type FS struct {
	zone *zone.Zone
	wd   Path
}

func (fs *FS) Zone() *zone.Zone {
	return fs.zone
}

func (fs *FS) WDPath() Path {
	return fs.wd
}

func (fs *FS) WD() (str string, id storage.ID, err error) {
	str = fs.wd.String()

	if len(fs.wd) > 0 {
		id = fs.wd[len(fs.wd)-1].ID
		return
	}

	var element tree.Element

	element, err = tree.DeserializeID(fs.zone.HeadID, fs.zone)
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

func (fs *FS) SetWD(path string) error {
	wd, err := fs.ResolvePath(path)
	if err != nil {
		return err
	}
	// Check that the last element is a directory.
	element, err := tree.DeserializeID(wd[len(wd)-1].ID, fs.zone)
	if err != nil {
		return err
	}
	_, ok := element.(*tree.Directory)
	if !ok {
		return fmt.Errorf("File '%s' is not a directory", path)
	}
	fs.wd = wd
	return nil
}

func (fs *FS) ResolvePath(filename string) (Path, error) {
	var parts []string

	if len(filename) > 0 {
		parts = file.PathSplit(filename)
	}

	var path Path
	if len(parts) == 0 || len(parts[0]) > 0 {
		// Relative path starting from the current working directory.
		path = fs.wd.Copy()
	} else {
		// Absolute path starting from the root.
		path = fs.wd[:1].Copy()
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
			entry, err := fs.LookupChild(path, part)
			if err != nil {
				return nil, err
			}
			path = append(path, *entry)
		}
	}

	return path, nil
}

func (fs *FS) LookupChild(path Path, name string) (*PathElement, error) {
	if len(path) == 0 {
		return nil, fmt.Errorf("No current working directory")
	}
	element, err := tree.DeserializeID(path[len(path)-1].ID, fs.zone)
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
