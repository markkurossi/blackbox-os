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
	"regexp"

	"github.com/markkurossi/blackbox-os/kernel/control"
	"github.com/markkurossi/blackbox-os/kernel/process"
)

func init() {
	builtin = append(builtin, Builtin{
		Name: "sysctl",
		Cmd:  cmd_sysctl,
	})
}

var reVar *regexp.Regexp = regexp.MustCompilePOSIX("^([^=]+)(=(.*))?$")

func cmd_sysctl(p *process.Process, args []string) {
	all := flag.Bool("a", false, "List all values.")
	flag.Parse()

	if *all {
		for _, value := range control.Values {
			fmt.Fprintf(p.Stdout, "%s\n", value)
		}
	} else if len(flag.Args()) > 0 {
		for _, v := range flag.Args() {
			matches := reVar.FindStringSubmatch(v)
			if matches == nil {
				fmt.Fprintf(p.Stderr, "Invalid command '%s'\n", v)
				return
			}
			if len(matches[3]) == 0 {
				val, err := control.Var(matches[1])
				if err != nil {
					fmt.Fprintf(p.Stderr, "%s\n", err)
					return
				}
				fmt.Fprintf(p.Stdout, "%s\n", val)
			} else {
				err := control.SetVar(matches[1], matches[3])
				if err != nil {
					fmt.Fprintf(p.Stderr, "%s\n", err)
				}
			}
		}
	} else {
		fmt.Fprintf(p.Stderr, "usage: %s name[=value]...\n", os.Args[0])
		fmt.Fprintf(p.Stderr, "       %s -a\n", os.Args[0])
	}
}
