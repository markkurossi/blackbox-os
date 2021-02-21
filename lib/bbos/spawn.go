//
// Copyright (c) 2021 Markku Rossi
//
// All rights reserved.
//

package bbos

import (
	"fmt"
)

func Spawn(argv []string, fds []int) (int, error) {
	var iargv []interface{}
	for _, arg := range argv {
		iargv = append(iargv, arg)
	}

	var ifds []interface{}
	for _, fd := range fds {
		ifds = append(ifds, fd)
	}

	data, err := Syscall("spawn", map[string]interface{}{
		"argv": iargv,
		"fds":  ifds,
	})
	if err != nil {
		return 0, err
	}
	pid, ok := data["ret"]
	if !ok {
		return 0, fmt.Errorf("Spawn: invalid response")
	}
	ipid, ok := pid.(int)
	if !ok {
		return 0, fmt.Errorf("Spawn: invalid response")
	}
	return ipid, nil
}

func Wait(pid int) (int, error) {
	data, err := Syscall("wait", map[string]interface{}{
		"pid": pid,
	})
	if err != nil {
		return 0, err
	}
	code, ok := data["ret"]
	if !ok {
		return 0, fmt.Errorf("Wait: invalid response")
	}
	icode, ok := code.(int)
	if !ok {
		return 0, fmt.Errorf("Wait: invalid response")
	}
	return icode, nil
}
