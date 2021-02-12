//
// Copyright (c) 2021 Markku Rossi
//
// All rights reserved.
//

package iface

import (
	"io"

	"github.com/markkurossi/blackbox-os/kernel/errno"
)

type FD interface {
	io.ReadWriteCloser
	Dup() FD
	Native() interface{}
}

var (
	_ FD = &FileDesc{}
)

type FileDesc struct {
	native   interface{}
	refCount int
}

func (fd *FileDesc) Read(p []byte) (n int, err error) {
	f, ok := fd.native.(io.Reader)
	if !ok {
		return 0, errno.EBADF
	}
	return f.Read(p)
}

func (fd *FileDesc) Write(p []byte) (n int, err error) {
	f, ok := fd.native.(io.Writer)
	if !ok {
		return 0, errno.EBADF
	}
	return f.Write(p)
}

func (fd *FileDesc) Close() error {
	f, ok := fd.native.(io.Closer)
	if !ok {
		return errno.EBADF
	}
	fd.refCount--
	if fd.refCount > 0 {
		return nil
	}
	return f.Close()
}

func (fd *FileDesc) Dup() FD {
	fd.refCount++
	return fd
}

func (fd *FileDesc) Native() interface{} {
	return fd.native
}

func NewFD(r interface{}) FD {
	return &FileDesc{
		native:   r,
		refCount: 1,
	}
}
