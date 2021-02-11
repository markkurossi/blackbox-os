//
// Copyright (c) 2021 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/markkurossi/blackbox-os/lib/bbos"
)

func main() {
	suppressNewline := flag.Bool("n", false, "suppress trailing newline")
	escapes := flag.Bool("e", false, "interpret backslash escapes")
	flag.Parse()

	fmt.Printf("os.Stdout: %#q\n", os.Stdout.Fd())

	flags, err := bbos.GetFlags(0)
	if err != nil {
		fmt.Printf("GetFlags failed: %s\n", err)
	} else {
		fmt.Printf("Stdin.Flags: 0x%x\n", flags)
	}

	var buf [1024]byte
	n, err := os.Stdin.Read(buf[:])
	if err != nil {
		fmt.Printf("read failed: %s\n", err)
	} else {
		os.Stdout.Write(buf[:n])
	}

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
