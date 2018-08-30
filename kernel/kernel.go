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

	"github.com/markkurossi/backup/lib/crypto/identity"
	"github.com/markkurossi/backup/lib/crypto/zone"
	"github.com/markkurossi/backup/lib/persistence"
	"github.com/markkurossi/blackbox-os/commands/shell"
	"github.com/markkurossi/blackbox-os/kernel/control"
	"github.com/markkurossi/blackbox-os/kernel/process"
	"github.com/markkurossi/blackbox-os/kernel/tty"
)

var (
	console = tty.NewConsole()
	IDs     []identity.PrivateKey
	FS      persistence.Accessor
	Zone    *zone.Zone
)

func main() {
	console.Flush()
	log.SetOutput(console)

	// Load identities.
	id, err := identity.GetNull()
	if err != nil {
		fmt.Fprintf(console, "Failed to load null identity: %s\n", err)
	} else {
		IDs = append(IDs, id)
	}

	// Init filesystem.
	FS, err = persistence.NewHTTP(control.FSRoot)
	if err != nil {
		fmt.Fprintf(console, "Failed to mount filesystem '%s': %s",
			control.FSRoot, err)
	}
	Zone, err = zone.Open(FS, control.FSZone, IDs)
	if err != nil {
		fmt.Fprintf(console, "Failed to open filesystem zone '%s': %s",
			control.FSZone, err)
	}

	fmt.Fprintf(console, "Black Box OS\n\n")
	fmt.Fprintf(console, "Type `help' for list of available commands.\n")

	shell.Shell(process.NewProcess(console, Zone))

	fmt.Fprintf(console, "\nSystem shutting down...\n")
}
