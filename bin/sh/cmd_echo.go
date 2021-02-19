//
// Copyright (c) 2021 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"fmt"
	"os"

	"github.com/markkurossi/blackbox-os/lib/bbos"
)

func init() {
	builtin = append(builtin, Builtin{
		Name: "echo",
		Cmd:  cmd_echo,
	})
}

func cmd_echo(args []string) {
	pid, err := bbos.Spawn(args, []int{
		int(os.Stdin.Fd()),
		int(os.Stdout.Fd()),
		int(os.Stderr.Fd()),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Spawn: %s\n", err)
		return
	}
	code, err := bbos.Wait(pid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Wait: %s\n", err)
		return
	}
	if code != 0 {
		fmt.Printf("exited: pid=%d, code=%d\n", pid, code)
	}
}
