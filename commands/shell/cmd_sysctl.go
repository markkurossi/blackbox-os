//
// cmd_sysctl.go
//
// Copyright (c) 2018 Markku Rossi
//
// All rights reserved.
//

package shell

import (
	"flag"
	"fmt"
	"os"

	"github.com/markkurossi/blackbox-os/kernel/control"
	"github.com/markkurossi/blackbox-os/kernel/process"
)

func init() {
	builtin = append(builtin, Builtin{
		Name: "sysctl",
		Cmd:  cmd_sysctl,
	})
}

func cmd_sysctl(p *process.Process, args []string) {
	all := flag.Bool("a", false, "List all values.")
	flag.Parse()

	if *all {
		for _, value := range control.Values {
			fmt.Fprintf(p.Stdout, "%s\n", value)
		}
	} else if len(flag.Args()) > 0 {
	} else {
		fmt.Fprintf(p.Stderr, "usage: %s name[=value]...\n", os.Args[0])
		fmt.Fprintf(p.Stderr, "       %s -a\n", os.Args[0])
	}
}
