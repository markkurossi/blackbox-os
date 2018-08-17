//
// kmsg.go
//
// Copyright (c) 2018 Markku Rossi
//
// All rights reserved.
//

package kmsg

import (
	"syscall/js"
)

var (
	kmsgPrint = js.Global().Get("kmsgPrint")
)

func Print(msg string) {
	kmsgPrint.Invoke(msg)
}
