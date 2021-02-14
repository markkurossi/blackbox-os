//
// Copyright (c) 2021 Markku Rossi
//
// All rights reserved.
//

package bbos

import (
	"fmt"
	"os"
)

func Open(name string) (*os.File, error) {
	data, err := Syscall("open", map[string]interface{}{
		"path": name,
	})
	if err != nil {
		return nil, err
	}
	val, ok := data["ret"]
	if !ok {
		return nil, fmt.Errorf("Open: invalid response")
	}
	fd, ok := val.(int)
	if !ok {
		return nil, fmt.Errorf("Open: invalid response")
	}
	return os.NewFile(uintptr(fd), name), nil
}
