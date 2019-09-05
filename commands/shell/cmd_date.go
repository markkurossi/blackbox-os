//
// cmd_date.go
//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package shell

import (
	"fmt"
	"time"

	"github.com/markkurossi/blackbox-os/kernel/process"
)

func init() {
	builtin = append(builtin, Builtin{
		Name: "date",
		Cmd:  cmd_date,
	})
}

func cmd_date(p *process.Process, args []string) {
	now := time.Now()
	fmt.Fprintf(p.Stdout, "%s\n", now.Format(time.UnixDate))
}
