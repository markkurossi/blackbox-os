//
// Copyright (c) 2021 Markku Rossi
//
// All rights reserved.
//

package bbos

import (
	"fmt"
	"io"
)

func Read(fd int, buf []byte) (int, error) {
	data, err := Syscall("read", map[string]interface{}{
		"fd":     fd,
		"length": len(buf),
	})
	if err != nil {
		return 0, err
	}
	val, ok := data["buf"]
	if !ok {
		return 0, fmt.Errorf("Read: invalid response")
	}
	bval, ok := val.([]byte)
	if !ok {
		return 0, fmt.Errorf("Read: invalid response")
	}
	if len(bval) == 0 {
		return 0, io.EOF
	}
	copy(buf, bval)
	return len(bval), nil
}

func Write(fd int, buf []byte) (int, error) {
	data, err := Syscall("write", map[string]interface{}{
		"fd":     fd,
		"data":   JSByteArray(buf),
		"offset": 0,
		"length": len(buf),
	})
	if err != nil {
		return 0, err
	}
	val, ok := data["ret"]
	if !ok {
		return 0, fmt.Errorf("Write: invalid response")
	}
	n, ok := val.(int)
	if !ok {
		return 0, fmt.Errorf("Write: invalid response")
	}
	if n == 0 {
		return 0, io.EOF
	}
	return n, nil
}

func Chdir(dir string) error {
	// XXX send path as string.
	data, err := Syscall("chdir", map[string]interface{}{
		"path": JSByteArray([]byte(dir)),
	})
	if err != nil {
		return err
	}
	val, ok := data["buf"]
	if !ok {
		return fmt.Errorf("Chdir: invalid response")
	}
	buf, ok := val.([]byte)
	if !ok {
		return fmt.Errorf("Chdir: invalid response")
	}
	SyscallSetWD(string(buf))

	return nil
}

func Getwd() (string, error) {
	data, err := Syscall("getwd", map[string]interface{}{})
	if err != nil {
		return "", err
	}
	val, ok := data["buf"]
	if !ok {
		return "", fmt.Errorf("Getwd: invalid response")
	}
	buf, ok := val.([]byte)
	if !ok {
		return "", fmt.Errorf("Getwd: invalid response")
	}
	wd := string(buf)
	SyscallSetWD(wd)

	return wd, nil
}
