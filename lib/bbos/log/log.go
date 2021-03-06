//
// Copyright (c) 2018-2021 Markku Rossi
//
// All rights reserved.
//

package log

import (
	"fmt"
	"io"
	"syscall/js"
)

var (
	console           = js.Global().Get("console")
	Writer  io.Writer = &writer{}
)

type writer struct {
}

func (w *writer) Write(p []byte) (n int, err error) {
	console.Call("log", string(p))
	return len(p), nil
}

func Print(msg string) {
	console.Call("log", msg)
}

func Printf(format string, a ...interface{}) {
	console.Call("log", fmt.Sprintf(format, a...))
}
