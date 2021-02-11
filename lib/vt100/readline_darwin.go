//
// Copyright (c) 2021 Markku Rossi
//
// All rights reserved.
//

package vt100

import (
	"io"
)

func MakeRaw(stdin io.Reader) (uint, error) {
	return 0, nil
}

func MakeCooked(stdin io.Reader, flags uint) error {
	return nil
}
