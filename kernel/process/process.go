//
// Copyright (c) 2018-2021 Markku Rossi
//
// All rights reserved.
//

package process

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"syscall/js"

	"github.com/markkurossi/backup/lib/crypto/zone"
	"github.com/markkurossi/blackbox-os/kernel/control"
	"github.com/markkurossi/blackbox-os/kernel/errno"
	"github.com/markkurossi/blackbox-os/kernel/fs"
	"github.com/markkurossi/blackbox-os/kernel/iface"
	"github.com/markkurossi/blackbox-os/kernel/kmsg"
	"github.com/markkurossi/blackbox-os/kernel/tty"
)

var (
	syscallSpawn  = js.Global().Get("syscallSpawn")
	syscallResult = js.Global().Get("syscallResult")
	uint8Array    = js.Global().Get("Uint8Array")
)

type Process struct {
	FDs    map[int]iface.FD
	FS     *fs.FS
	nextFD int
}

func New(stdin, stdout, stderr iface.FD, z *zone.Zone) (*Process, error) {
	fs, err := fs.New(z)
	if err != nil {
		return nil, err
	}
	p := &Process{
		FDs:    make(map[int]iface.FD),
		FS:     fs,
		nextFD: 3,
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

func (p *Process) NextFD() int {
	fd := p.nextFD
	p.nextFD++
	return fd
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
	idVal := event.Get("id")
	if idVal.IsNull() || idVal.IsUndefined() {
		kmsg.Printf("syscall: no call ID")
		return
	}
	id := idVal.Int()
	err := p.syscallHandler(id, worker, event)
	if err != nil {
		syscallResult.Invoke(worker, id, err.Error())
	}
}

func (p *Process) syscallHandler(id int, worker, event js.Value) error {
	switch event.Get("cmd").String() {
	case "open":
		filename, err := getString(event, "path")
		if err != nil {
			return err
		}
		f, err := fs.Open(p.FS, string(filename))
		if err != nil {
			kmsg.Printf("syscall: open: %s", err)
			return errno.EINVAL
		}
		fd := p.NextFD()
		p.FDs[fd] = iface.NewFD(f.Reader())
		syscallResult.Invoke(worker, id, nil, fd)

	case "write":
		fd := event.Get("fd").Int()
		data, err := getData(event, "data")
		if err != nil {
			return err
		}
		offset := event.Get("offset").Int()
		length := event.Get("length").Int()

		if offset < 0 || offset+length > len(data) {
			return errno.EINVAL
		}

		f, ok := p.FDs[fd]
		if !ok {
			return errno.EBADF
		}

		n, err := f.Write(data[offset : offset+length])
		if err != nil {
			return err
		}

		syscallResult.Invoke(worker, id, nil, n)

	case "read":
		fd := event.Get("fd").Int()
		length := event.Get("length").Int()

		f, ok := p.FDs[fd]
		if !ok {
			return errno.EBADF
		}

		data := make([]byte, length)
		n, err := f.Read(data)
		if err != nil {
			if err == io.EOF {
				syscallResult.Invoke(worker, id, nil, 0)
				return nil
			}
			return err
		}

		buf := uint8Array.New(n)
		js.CopyBytesToJS(buf, data[:n])
		syscallResult.Invoke(worker, id, nil, n, buf)

	case "ioctl":
		fd := event.Get("fd").Int()
		switch event.Get("request").String() {
		case "GetFlags":
			f, ok := p.FDs[fd]
			if !ok {
				return errno.EBADF
			}
			var flags int
			switch native := f.Native().(type) {
			case *tty.Console:
				flags = int(native.Flags())

			default:
				return errno.EBADF
			}
			syscallResult.Invoke(worker, id, nil, flags)

		case "SetFlags":
			f, ok := p.FDs[fd]
			if !ok {
				return errno.EBADF
			}
			flags := event.Get("value").Int()

			switch native := f.Native().(type) {
			case *tty.Console:
				native.SetFlags(tty.TTYFlags(flags))

			default:
				return errno.EBADF
			}
			syscallResult.Invoke(worker, id, nil, 0)

		default:
			kmsg.Printf("syscall ioctl: %s not implemented yet\n",
				event.Get("request").String())
			return errno.ENOSYS
		}

	case "chdir":
		path, err := getData(event, "path")
		if err != nil {
			return err
		}
		err = p.FS.SetWD(string(path))
		if err != nil {
			return err
		}
		fallthrough

	case "getwd":
		wd, _, err := p.FS.WD()
		if err != nil {
			return err
		}
		data := []byte(wd)

		buf := uint8Array.New(len(data))
		js.CopyBytesToJS(buf, data)
		syscallResult.Invoke(worker, id, nil, len(data), buf)

	default:
		kmsg.Printf("syscall: %s: not implemented\n",
			event.Get("cmd").String())
		return errno.ENOSYS
	}

	return nil
}

func getData(event js.Value, name string) ([]byte, error) {
	val := event.Get(name)
	if val.IsNull() || val.IsUndefined() {
		return nil, errno.EINVAL
	}

	buf := make([]byte, val.Length())
	js.CopyBytesToGo(buf, val)

	return buf, nil
}

func getString(event js.Value, name string) (string, error) {
	val := event.Get(name)
	switch val.Type() {
	case js.TypeString:
		return val.String(), nil

	default:
		return "", errno.EINVAL
	}
}
