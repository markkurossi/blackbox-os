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
	kmsgPrint = js.Global().Get("kmsgPrint")
)

func Print(msg string) {
	kmsgPrint.Invoke(msg)
}

func Printf(format string, a ...interface{}) {
	kmsgPrint.Invoke(fmt.Sprintf(format, a...))
}
