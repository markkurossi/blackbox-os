//
// kernel.go
//
// Copyright (c) 2018-2021 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"fmt"
	"io"
	"log"

	"github.com/markkurossi/backup/lib/crypto/identity"
	"github.com/markkurossi/backup/lib/crypto/zone"
	"github.com/markkurossi/backup/lib/persistence"
	"github.com/markkurossi/blackbox-os/kernel/control"
	"github.com/markkurossi/blackbox-os/kernel/fs"
	"github.com/markkurossi/blackbox-os/kernel/iface"
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
	parseParams()

	console.Flush()
	log.SetOutput(console)
	err := runInit()
	if err != nil {
		fmt.Fprintf(console, "Init failed: %s\n", err)
	}

	fmt.Fprintf(console, "\nSystem halted.\n")
}

func runInit() error {
	// Load identities.
	id, err := identity.GetNull()
	if err != nil {
		return fmt.Errorf("Failed to load null identity: %s", err)
	}
	IDs = append(IDs, id)

	// Init filesystem.
	FS, err = persistence.NewHTTP(control.FSRoot)
	if err != nil {
		return fmt.Errorf("Failed to mount filesystem '%s': %s",
			control.FSRoot, err)
	}
	Zone, err = zone.Open(FS, control.FSZone, IDs)
	if err != nil {
		return fmt.Errorf("Failed to open filesystem zone '%s': %s",
			control.FSZone, err)
	}

	// Run init.
	process, err := process.New(iface.NewFD(console), iface.NewFD(console),
		iface.NewFD(console), Zone)
	if err != nil {
		return fmt.Errorf("Failed to create init process: %s", err)
	}
	motd, err := fs.Open(process.FS, "/etc/motd")
	if err != nil {
		fmt.Fprintf(console, "Black Box OS\n\n")
	} else {
		io.Copy(console, motd.Reader())
	}

	fmt.Fprintf(console, "\nType `help' for list of available commands.\n")
	err = process.Run("sh", []string{})
	if err != nil {
		return err
	}
	return nil
}
