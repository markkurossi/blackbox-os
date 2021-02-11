//
// Copyright (c) 2021 Markku Rossi
//
// All rights reserved.
//

package bbos

import (
	"errors"
	"syscall/js"
)

var (
	syscall = js.Global().Get("syscall")
)

func Syscall(call string, fd int, params map[string]interface{}) (
	map[string]interface{}, error) {

	params["type"] = call
	params["fd"] = fd

	c := make(chan []js.Value)

	ctx := map[string]interface{}{
		"cb": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			c <- args
			return nil
		}),
	}

	syscall.Invoke(js.ValueOf(params), js.ValueOf(ctx))

	result := <-c

	if !result[0].IsNull() {
		return nil, errors.New(result[0].Get("code").String())
	}

	return map[string]interface{}{
		"Flags": result[1].Int(),
	}, nil
}
