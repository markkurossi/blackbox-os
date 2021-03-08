//
// Copyright (c) 2021 Markku Rossi
//
// All rights reserved.
//

package readline

import (
	"io"
)

// MakeRaw enables raw input and disables echo.
func MakeRaw(stdin io.Reader) (uint, error) {
	return 0, nil
}

// MakeCooked enables the input mode based on flags.
func MakeCooked(stdin io.Reader, flags uint) error {
	return nil
}
