//
// Copyright (c) 2021 Markku Rossi
//
// All rights reserved.
//

package bbos

import (
	"fmt"
)

func GetFlags(fd int) (int, error) {
	data, err := Syscall("ioctl", map[string]interface{}{
		"fd":      fd,
		"request": "GetFlags",
	})
	if err != nil {
		return 0, err
	}
	flags, ok := data["ret"]
	if !ok {
		return 0, fmt.Errorf("GetFlags: invalid response")
	}
	iflags, ok := flags.(int)
	if !ok {
		return 0, fmt.Errorf("GetFlags: invalid response")
	}
	return iflags, nil
}

func SetFlags(fd, flags int) error {
	_, err := Syscall("ioctl", map[string]interface{}{
		"fd":      fd,
		"request": "SetFlags",
		"value":   flags,
	})
	return err
}
