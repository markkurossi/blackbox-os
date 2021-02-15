//
// Copyright (c) 2021 Markku Rossi
//
// All rights reserved.
//

package bbos

import (
	"fmt"
)

func Chdir(dir string) error {
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
