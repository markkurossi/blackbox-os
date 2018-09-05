//
// tabulate.go
//
// Copyright (c) 2018 Markku Rossi
//
// All rights reserved.
//

package emulator

import (
	"fmt"
	"io"
)

func Tabulate(items []string, out io.Writer) {
	var max = 0

	// Count the length of the longest element.
	for _, i := range items {
		len := len(i)
		if len > max {
			max = len
		}
	}

	width := (max/8 + 1) * 8
	perLine := 80 / width
	if perLine < 1 {
		perLine = 1
	}

	count := 0

	for _, i := range items {
		fmt.Fprintf(out, "%s", i)
		count++
		if count >= perLine {
			fmt.Fprintf(out, "\n")
			count = 0
		} else {
			len := len(i)
			len = (len/8 + 1) * 8
			fmt.Fprintf(out, "\t")

			for len < width {
				fmt.Fprintf(out, "\t")
				len += 8
			}
		}
	}
	if count > 0 {
		fmt.Fprintf(out, "\n")
	}
}
