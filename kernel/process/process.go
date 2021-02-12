//
// Copyright (c) 2018-2021 Markku Rossi
//
// All rights reserved.
//

package process

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	"syscall/js"

	"github.com/markkurossi/backup/lib/crypto/zone"
	"github.com/markkurossi/blackbox-os/kernel/control"
	"github.com/markkurossi/blackbox-os/kernel/errno"
	"github.com/markkurossi/blackbox-os/kernel/fs"
	"github.com/markkurossi/blackbox-os/kernel/iface"
	"github.com/markkurossi/blackbox-os/kernel/kmsg"
)

var (
	syscallSpawn  = js.Global().Get("syscallSpawn")
	syscallResult = js.Global().Get("syscallResult")
	uint8Array    = js.Global().Get("Uint8Array")
)

type Process struct {
	FDs map[int]iface.FD
	FS  *fs.FS
}

func New(stdin, stdout, stderr iface.FD, z *zone.Zone) (*Process, error) {
	fs, err := fs.New(z)
	if err != nil {
		return nil, err
	}
	p := &Process{
		FDs: make(map[int]iface.FD),
		FS:  fs,
	}

	if stdin != nil {
		p.FDs[0] = stdin
	}
	if stdout != nil {
		p.FDs[1] = stdout
	}
	if stderr != nil {
		p.FDs[2] = stderr
	}

	return p, nil
}

func (p *Process) Run(cmd string, args []string) error {
	var worker js.Value

	c := make(chan error)

	onSyscall := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) != 1 {
			kmsg.Printf("syscall: invalid arguments: %v\n", args)
			return nil
		}
		go p.syscall(worker, args[0])
		return nil
	})
	// XXX onError

	resp, err := http.Get(fmt.Sprintf("%s/bin/%s.wasm", control.BaseURL, cmd))
	if err != nil {
		return fmt.Errorf("process: load %v: %w", cmd, err)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("process: read data: %w", err)
	}

	code := uint8Array.New(len(data))
	js.CopyBytesToJS(code, data)

	argv := []interface{}{
		onSyscall, code, cmd,
	}
	for _, arg := range args {
		argv = append(argv, arg)
	}

	worker = syscallSpawn.Invoke(argv...)

	return <-c
}

func (p *Process) syscall(worker, event js.Value) {
	var id int
	idVal := event.Get("id")
	if !idVal.IsNull() {
		id = idVal.Int()
	}

	switch event.Get("type").String() {
	case "write":
		fd := event.Get("fd").Int()
		dval := event.Get("data")
		offset := event.Get("offset").Int()
		length := event.Get("length").Int()

		data := make([]byte, dval.Length())
		js.CopyBytesToGo(data, dval)

		if offset < 0 || offset+length > len(data) {
			kmsg.Printf("syscall write: id=%d, fd=%d, offset=%d, length=%d",
				id, fd, offset, length)
			syscallResult.Invoke(worker, id, errno.EINVAL.Error(), 0)
			return
		}

		f, ok := p.FDs[fd]
		if !ok {
			syscallResult.Invoke(worker, id, errno.EBADF.Error(), 0)
			return
		}

		n, err := f.Write(data[offset : offset+length])
		if err != nil {
			kmsg.Printf("syscall write: id=%d, fd=%d => %d %s:\n%s",
				id, fd, n, err, hex.Dump(data))
			syscallResult.Invoke(worker, id, err.Error(), n)
			return
		}

		kmsg.Printf("syscall write: id=%d, fd=%d => %d:\n%s",
			id, fd, n, hex.Dump(data))
		syscallResult.Invoke(worker, id, nil, n)

	case "read":
		fd := event.Get("fd").Int()
		length := event.Get("length").Int()

		f, ok := p.FDs[fd]
		if !ok {
			syscallResult.Invoke(worker, id, errno.EBADF.Error(), 0)
			return
		}

		data := make([]byte, length)
		_, err := f.Read(data)
		if err != nil {
			syscallResult.Invoke(worker, id, err.Error(), 0)
			return
		}

		kmsg.Printf("syscall read: id=%d, fd=%d => %d:\n%s",
			id, fd, len(data), hex.Dump(data))

		buf := uint8Array.New(len(data))
		js.CopyBytesToJS(buf, data)
		syscallResult.Invoke(worker, id, nil, len(data), buf)

	case "ioctl":
		kmsg.Printf("syscall ioctl: not implemented yet\n")
		syscallResult.Invoke(worker, id, nil, 0)

	default:
		kmsg.Printf("syscall: type=%v\n", event.Get("type").String())
		syscallResult.Invoke(worker, id, errno.ENOSYS.Error(), 0)
	}
}
