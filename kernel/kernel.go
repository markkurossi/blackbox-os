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
	"net/url"
	"syscall/js"

	"github.com/markkurossi/backup/lib/crypto/identity"
	"github.com/markkurossi/backup/lib/crypto/zone"
	"github.com/markkurossi/backup/lib/persistence"
	"github.com/markkurossi/blackbox-os/commands/shell"
	"github.com/markkurossi/blackbox-os/kernel/control"
	"github.com/markkurossi/blackbox-os/kernel/process"
	"github.com/markkurossi/blackbox-os/kernel/tty"
)

var (
	console     = tty.NewConsole()
	IDs         []identity.PrivateKey
	FS          persistence.Accessor
	Zone        *zone.Zone
	locationURL = js.Global().Get("location").Get("href").String()
)

func main() {
	parseParams()

	console.Flush()
	log.SetOutput(console)
	runInit()

	fmt.Fprintf(console, "\nSystem halted.\n")
}

func runInit() {
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
		fmt.Fprintf(console, "Failed to mount filesystem '%s': %s\n",
			control.FSRoot, err)
		return
	}
	Zone, err = zone.Open(FS, control.FSZone, IDs)
	if err != nil {
		fmt.Fprintf(console, "Failed to open filesystem zone '%s': %s\n",
			control.FSZone, err)
		return
	}

	fmt.Fprintf(console, "Black Box OS\n\n")
	fmt.Fprintf(console, "Type `help' for list of available commands.\n")

	process, err := process.NewProcess(console, Zone)
	if err != nil {
		fmt.Fprintf(console, "Failed to create init process: %s\n", err)
	} else {
		shell.Shell(process)
	}
}

func parseParams() {
	url, err := url.Parse(locationURL)
	if err != nil {
		fmt.Fprintf(console, "Failed to parse location URL '%s': %s\n",
			locationURL, err)
	}
	url.RawQuery = ""
	url.Fragment = ""

	control.FSRoot = fmt.Sprintf("%s/fs", url)
}
