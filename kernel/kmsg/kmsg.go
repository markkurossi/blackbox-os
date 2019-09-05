//
// kmsg.go
//
// Copyright (c) 2018-2019 Markku Rossi
//
// All rights reserved.
//

package kmsg

import (
	"fmt"
	"syscall/js"
)

var (
	console = js.Global().Get("console")
)

func Print(msg string) {
	console.Call("log", msg)
}

func Printf(format string, a ...interface{}) {
	console.Call("log", fmt.Sprintf(format, a...))
}
