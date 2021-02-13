//
// Copyright (c) 2021 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"flag"
	"fmt"
)

func main() {
	suppressNewline := flag.Bool("n", false, "suppress trailing newline")
	escapes := flag.Bool("e", false, "interpret backslash escapes")
	flag.Parse()

	_ = escapes

	for idx, arg := range flag.Args() {
		if idx > 0 {
			fmt.Print(" ")
		}
		fmt.Print(arg)
	}
	if !*suppressNewline {
		fmt.Println()
	}
}
