//
// Copyright (c) 2018-2021 Markku Rossi
//
// All rights reserved.
//

package vt100

import (
	"fmt"
	"os"
)

// ReadPassword reads a password.
func ReadPassword(prompt string) (string, error) {
	rl := NewReadline(os.Stdin, os.Stdout, os.Stderr)
	rl.Mask = MaskAsterisk
	password, err := rl.Read(prompt)
	fmt.Fprintln(os.Stdout)
	return password, err
}
