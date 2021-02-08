//
// Copyright (c) 2021 Markku Rossi
//
// All rights reserved.
//

package shell

import (
	"syscall/js"

	"github.com/markkurossi/blackbox-os/kernel/process"
)

var (
	spawn = js.Global().Get("spawn")
)

func init() {
	builtin = append(builtin, Builtin{
		Name: "spawn",
		Cmd:  cmd_spawn,
	})
}

func cmd_spawn(p *process.Process, args []string) {
	spawn.Invoke()
}
