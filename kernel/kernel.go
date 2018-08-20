//
// kernel.go
//
// Copyright (c) 2018 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"fmt"
	"log"

	"github.com/markkurossi/blackbox-os/commands/shell"
	"github.com/markkurossi/blackbox-os/kernel/process"
	"github.com/markkurossi/blackbox-os/kernel/tty"
)

var console = tty.NewConsole()

func main() {
	console.Flush()
	log.SetOutput(console)

	fmt.Fprintf(console, "Black Box OS\n\n")
	fmt.Fprintf(console, "Type `help' for list of available commands.\n")

	shell.Shell(process.NewProcess(console))

	fmt.Fprintf(console, "\nSystem shutting down...\n")
}
