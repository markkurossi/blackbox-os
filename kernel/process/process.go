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
	"sync"
	"syscall/js"

	"github.com/markkurossi/backup/lib/crypto/zone"
	"github.com/markkurossi/backup/lib/tree"
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

var (
	byID   = make(map[int]*Process)
	nextID = 0
)

type Process struct {
	ID       int
	mutex    sync.Mutex
	cond     *sync.Cond
	exited   bool
	exitCode int
	FDs      map[int]iface.FD
	FS       *fs.FS
	nextFD   int
}

func New(stdin, stdout, stderr iface.FD, z *zone.Zone) (*Process, error) {
	fs, err := fs.New(z)
	if err != nil {
		return nil, err
	}
	p := &Process{
		ID:     nextID,
		FDs:    make(map[int]iface.FD),
		FS:     fs,
		nextFD: 3,
	}
	nextID++
	p.cond = sync.NewCond(&p.mutex)

	if stdin != nil {
		p.FDs[0] = stdin
	}
	if stdout != nil {
		p.FDs[1] = stdout
	}
	if stderr != nil {
		p.FDs[2] = stderr
	}

	byID[p.ID] = p

	return p, nil
}

func (p *Process) Exit(code int) {
	p.cond.L.Lock()

	p.exitCode = code
	p.exited = true
	p.cond.Signal()

	p.cond.L.Unlock()
}

func (p *Process) Wait() int {
	p.cond.L.Lock()
	for !p.exited {
		p.cond.Wait()
	}
	p.cond.L.Unlock()
	return p.exitCode
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
		go p.syscall(c, worker, args[0])
		return nil
	})
	onError := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		var message string
		if len(args) != 1 {
			kmsg.Printf("onerror: invalid arguments: %v\n", args)
			message = "unknown error"
		} else {
			message = args[0].String()
		}
		c <- fmt.Errorf("onerror: %s", message)
		return nil
	})

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
		onSyscall, onError, p.ID, code, cmd,
	}
	for _, arg := range args {
		argv = append(argv, arg)
	}

	worker = syscallSpawn.Invoke(argv...)

	return <-c
}

func (p *Process) syscall(c chan error, worker, event js.Value) {
	idVal := event.Get("id")
	if idVal.IsNull() || idVal.IsUndefined() {
		kmsg.Printf("syscall: no call ID")
		return
	}
	id := idVal.Int()
	err := p.syscallHandler(c, id, worker, event)
	if err != nil {
		syscallResult.Invoke(worker, id, err.Error())
	}
}

func (p *Process) syscallHandler(c chan error, id int, worker,
	event js.Value) error {

	switch event.Get("cmd").String() {
	case "open":
		filename, err := getString(event, "path")
		if err != nil {
			return err
		}
		f, err := fs.Open(p.FS, filename)
		if err != nil {
			kmsg.Printf("syscall: open: %s", err)
			return errno.EINVAL
		}
		fd := p.NextFD()
		p.FDs[fd] = iface.NewFD(f.Reader())
		syscallResult.Invoke(worker, id, nil, fd)

	case "write":
		f, err := p.getFD(event)
		if err != nil {
			return err
		}
		data, err := getData(event, "data")
		if err != nil {
			return err
		}
		offset, err := getInt(event, "offset")
		if err != nil {
			return err
		}
		length, err := getInt(event, "length")
		if err != nil {
			return err
		}

		if offset < 0 || offset+length > len(data) {
			return errno.EINVAL
		}

		n, err := f.Write(data[offset : offset+length])
		if err != nil {
			return err
		}

		syscallResult.Invoke(worker, id, nil, n)

	case "read":
		f, err := p.getFD(event)
		if err != nil {
			return err
		}
		length, err := getInt(event, "length")
		if err != nil {
			return err
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

	case "fstat":
		f, err := p.getFD(event)
		if err != nil {
			return err
		}
		info, err := p.stat(f.Native())
		if err != nil {
			return err
		}
		syscallResult.Invoke(worker, id, nil, 0, nil, js.ValueOf(info))

	case "stat":
		path, err := getString(event, "path")
		if err != nil {
			return err
		}
		info, err := p.stat(path)
		if err != nil {
			return err
		}
		syscallResult.Invoke(worker, id, nil, 0, nil, js.ValueOf(info))

	case "ioctl":
		f, err := p.getFD(event)
		if err != nil {
			return err
		}
		switch event.Get("request").String() {
		case "GetFlags":
			var flags int
			switch native := f.Native().(type) {
			case *tty.Console:
				flags = int(native.Flags())

			default:
				return errno.EBADF
			}
			syscallResult.Invoke(worker, id, nil, flags)

		case "SetFlags":
			flags, err := getInt(event, "value")
			if err != nil {
				return err
			}

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

	case "readdir":
		path, err := getString(event, "path")
		if err != nil {
			return err
		}
		info, err := fs.ReadDir(p.FS, path)
		if err != nil {
			kmsg.Printf("syscall: readdir: %s", err)
			return errno.EINVAL
		}
		var names []interface{}
		for _, fi := range info {
			names = append(names, fi.Name())
		}
		syscallResult.Invoke(worker, id, nil, 0, nil, js.ValueOf(names))

	case "spawn":
		argv, err := getStringArray(event, "argv")
		if err != nil {
			return err
		}
		if len(argv) == 0 {
			return errno.EINVAL
		}
		fds, err := getIntArray(event, "fds")
		if err != nil {
			return errno.EINVAL
		}
		process, err := New(nil, nil, nil, p.FS.Zone())
		if err != nil {
			return errno.EINVAL
		}

		for idx, fd := range fds {
			f, ok := p.FDs[fd]
			if !ok {
				return errno.EINVAL
			}
			process.FDs[idx] = f.Dup()
		}

		go func() {
			err := process.Run(argv[0], argv[1:])
			if err != nil {
				fmt.Printf("process terminated: %v\n", err)
			}
		}()
		syscallResult.Invoke(worker, id, nil, process.ID)

	case "wait":
		pid, err := getInt(event, "pid")
		if err != nil {
			return err
		}
		process, ok := byID[pid]
		if !ok {
			return errno.ENOENT
		}
		code := process.Wait()
		syscallResult.Invoke(worker, id, nil, code)

	case "exit":
		code, err := getInt(event, "code")
		if err != nil {
			return err
		}
		p.Exit(code)
		syscallResult.Invoke(worker, id, nil, 0)
		c <- nil

	default:
		kmsg.Printf("syscall: %s: not implemented\n",
			event.Get("cmd").String())
		return errno.ENOSYS
	}

	return nil
}

func (p *Process) getFD(event js.Value) (iface.FD, error) {
	fd, err := getInt(event, "fd")
	if err != nil {
		return nil, err
	}
	f, ok := p.FDs[fd]
	if !ok {
		return nil, errno.EBADF
	}
	return f, nil
}

func getInt(event js.Value, name string) (int, error) {
	val := event.Get(name)
	switch val.Type() {
	case js.TypeNumber:
		return val.Int(), nil

	default:
		return 0, errno.EINVAL
	}
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

func getStringArray(event js.Value, name string) ([]string, error) {
	val := event.Get(name)
	switch val.Type() {
	case js.TypeObject:
		result := make([]string, val.Length())
		for i := 0; i < len(result); i++ {
			v := val.Index(i)
			if v.Type() != js.TypeString {
				return nil, errno.EINVAL
			}
			result[i] = v.String()
		}
		return result, nil

	default:
		return nil, errno.EINVAL
	}
}

func getIntArray(event js.Value, name string) ([]int, error) {
	val := event.Get(name)
	switch val.Type() {
	case js.TypeObject:
		result := make([]int, val.Length())
		for i := 0; i < len(result); i++ {
			v := val.Index(i)
			if v.Type() != js.TypeNumber {
				return nil, errno.EINVAL
			}
			result[i] = v.Int()
		}
		return result, nil

	default:
		return nil, errno.EINVAL
	}
}

func (p *Process) stat(native interface{}) (map[string]interface{}, error) {
	result := map[string]interface{}{
		"dev":     0,
		"ino":     0,
		"mode":    0,
		"nlink":   0,
		"uid":     0,
		"gid":     0,
		"rdev":    0,
		"size":    0,
		"blksize": 0,
		"blocks":  0,
		"atimeMs": 0,
		"mtimeMs": 0,
		"ctimeMs": 0,
	}
	_ = result

	switch handle := native.(type) {
	case *fs.File:
		switch h := handle.Handle.(type) {
		case *tree.Directory:
			result["mode"] = fs.S_IFDIR
			return result, nil

		default:
			kmsg.Printf("stat: invalid file: %T", h)
			return nil, errno.EINVAL
		}

	case *tree.SimpleReader:
		result["size"] = int(handle.Size())
		result["mode"] = fs.S_IFREG
		return result, nil

	case string:
		info, err := fs.Stat(p.FS, handle)
		if err != nil {
			kmsg.Printf("stat: %s: %s", handle, err)
			return nil, errno.ENOENT
		}
		if info.IsDir() {
			result["mode"] = fs.S_IFDIR
		} else {
			result["mode"] = fs.S_IFREG
		}
		result["size"] = int(info.Size())
		return result, nil

	default:
		kmsg.Printf("stat: invalid handle: %T", handle)
		return nil, errno.EINVAL
	}
}
