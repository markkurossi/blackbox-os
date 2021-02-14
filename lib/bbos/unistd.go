//
// Copyright (c) 2021 Markku Rossi
//
// All rights reserved.
//

package bbos

import (
	"fmt"
)

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
	return string(buf), nil
}

func Chdir(dir string) error {
	_, err := Syscall("chdir", map[string]interface{}{
		"data": JSByteArray([]byte(dir)),
	})
	return err
}
