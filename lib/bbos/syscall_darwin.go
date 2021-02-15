//
// Copyright (c) 2021 Markku Rossi
//
// All rights reserved.
//

package bbos

import (
	"fmt"
)

func JSByteArray(data []byte) []byte {
	return data
}

func Syscall(call string, params map[string]interface{}) (
	map[string]interface{}, error) {

	return nil, fmt.Errorf("Syscall not implemented")
}

func SyscallSetWD(cwd string) {
}
