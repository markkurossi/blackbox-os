//
// filesystem.go
//
// Copyright (c) 2018-2021 Markku Rossi
//
// All rights reserved.
//

package fs

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/markkurossi/backup/lib/tree"
)

type FileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
	element tree.Element
}

func (info *FileInfo) Name() string {
	return info.name
}

func (info *FileInfo) Size() int64 {
	return info.size
}

func (info *FileInfo) Mode() os.FileMode {
	return info.mode
}

func (info *FileInfo) ModTime() time.Time {
	return info.modTime
}

func (info *FileInfo) IsDir() bool {
	return (info.mode & os.ModeDir) != 0
}

func (info *FileInfo) Sys() interface{} {
	return info.element
}

func Stat(fs *FS, name string) (os.FileInfo, error) {
	path, err := fs.ResolvePath(name)
	if err != nil {
		return nil, err
	}
	element, err := tree.DeserializeID(path[len(path)-1].ID, fs.Zone())
	if err != nil {
		return nil, err
	}
	switch el := element.(type) {
	case *tree.Directory:
		return &FileInfo{
			name:    path[len(path)-1].Name,
			mode:    os.ModeDir,
			element: element,
		}, nil

	case tree.File:
		return &FileInfo{
			name:    path[len(path)-1].Name,
			size:    el.Size(),
			element: element,
		}, nil

	default:
		return nil, fmt.Errorf("Invalid element %T", element)
	}
}

func ReadDir(fs *FS, dirname string) ([]os.FileInfo, error) {
	info, err := Stat(fs, dirname)
	if err != nil {
		return nil, err
	}
	dir, ok := info.Sys().(*tree.Directory)
	if !ok {
		return nil, fmt.Errorf("File '%s' is not a directory", dirname)
	}

	// XXX resolve path twice: here and Stat above
	path, err := fs.ResolvePath(dirname)
	if err != nil {
		return nil, err
	}
	dirName := path.String()

	var result []os.FileInfo
	for _, entry := range dir.Entries {
		i, err := Stat(fs, fmt.Sprintf("%s/%s", dirName, entry.Name))
		if err != nil {
			return nil, err
		}
		ii, ok := i.(*FileInfo)
		if ok {
			ii.modTime = time.Unix(0, entry.ModTime)
		}
		result = append(result, i)
	}

	return result, nil
}

type File struct {
	handle tree.File
}

func (f *File) Reader() io.Reader {
	return f.handle.Reader()
}

func Open(fs *FS, name string) (*File, error) {
	path, err := fs.ResolvePath(name)
	if err != nil {
		return nil, err
	}

	element, err := tree.DeserializeID(path[len(path)-1].ID, fs.Zone())
	if err != nil {
		return nil, err
	}
	file, ok := element.(tree.File)
	if !ok {
		return nil, fmt.Errorf("Not a regular file: %s", name)
	}
	return &File{
		handle: file,
	}, nil
}
