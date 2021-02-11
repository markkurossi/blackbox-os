//
// Copyright (c) 2021 Markku Rossi
//
// All rights reserved.
//

package bbos

import (
	"fmt"
)

func Syscall(call string, fd int, params map[string]interface{}) (
	map[string]interface{}, error) {

	return nil, fmt.Errorf("Syscall not implemented")
}
