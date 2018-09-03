//
// cmd_http.go
//
// Copyright (c) 2018 Markku Rossi
//
// All rights reserved.
//

package shell

import (
	"fmt"

	"github.com/markkurossi/backup/lib/crypto/identity"
	"github.com/markkurossi/backup/lib/crypto/zone"
	"github.com/markkurossi/backup/lib/objtree"
	"github.com/markkurossi/backup/lib/persistence"
	"github.com/markkurossi/blackbox-os/kernel/process"
)

func init() {
	builtin = append(builtin, Builtin{
		Name: "http",
		Cmd:  cmd_http,
	})
}

func cmd_http(p *process.Process, args []string) {
	null, err := identity.GetNull()
	if err != nil {
		fmt.Printf("Failed to get null ID: %s\n", err)
		return
	}

	root, err := persistence.NewHTTP("http://localhost:8100/fs/.backup")
	if err != nil {
		fmt.Printf("HTTP error: %s\n", err)
		return
	}
	z, err := zone.Open(root, "default", []identity.PrivateKey{null})
	if err != nil {
		fmt.Printf("HTTP error: %s\n", err)
		return
	}
	err = objtree.List(z.HeadID, z, true)
	if err != nil {
		fmt.Printf("HTTP error: %s\n", err)
		return
	}
}
