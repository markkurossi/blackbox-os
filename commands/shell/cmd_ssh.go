//
// cmd_ssh.go
//
// Copyright (c) 2018 Markku Rossi
//
// All rights reserved.
//

package shell

import (
	"encoding/hex"
	"fmt"
	"regexp"
	"time"

	"github.com/markkurossi/blackbox-os/kernel/network"
	"github.com/markkurossi/blackbox-os/kernel/process"
)

var reTarget *regexp.Regexp = regexp.MustCompilePOSIX("([^@]+@)?([^:]+)(:.*)?")

func cmd_ssh(p *process.Process, args []string) {
	if len(args) < 2 {
		fmt.Fprintf(p.Stdout, "Usage: ssh [user@]host[:port]\n")
		return
	}

	matches := reTarget.FindStringSubmatch(args[1])
	if matches == nil {
		fmt.Fprintf(p.Stderr, "Invalid target '%s'\n", args[1])
		return
	}

	user := matches[1]
	host := matches[2]
	port := matches[3]

	if len(port) == 0 {
		port = ":22"
	}
	addr := host + port

	fmt.Fprintf(p.Stdout, "Connecting to %s@%s...\n", user, addr)

	conn, err := network.DialTimeout(addr, 5*time.Second)
	if err != nil {
		fmt.Fprintf(p.Stderr, "Dial failed: %s\n", err)
		return
	}

	go func() {
		var buf [1024]byte
		for {
			n, err := conn.Read(buf[:])
			if err != nil {
				break
			}
			fmt.Fprintf(p.Stdout, "conn:\n%s", hex.Dump(buf[:n]))

		}
		fmt.Fprintf(p.Stdout, "Connection to %s closed\n", addr)
		conn.Close()
	}()
}
