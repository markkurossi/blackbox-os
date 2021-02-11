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
	data, err := Syscall("ioctl", fd, map[string]interface{}{
		"request": "GetFlags",
	})
	if err != nil {
		return 0, err
	}
	flags, ok := data["Flags"]
	if !ok {
		return 0, fmt.Errorf("GetFlags: invalid response")
	}
	iflags, ok := flags.(int)
	if !ok {
		return 0, fmt.Errorf("GetFlags: invalid response")
	}
	return iflags, nil
}
