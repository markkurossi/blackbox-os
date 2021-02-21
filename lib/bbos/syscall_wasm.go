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
	syscall      = js.Global().Get("syscall")
	syscallSetWD = js.Global().Get("syscallSetWD")
	uint8Array   = js.Global().Get("Uint8Array")
)

func JSByteArray(data []byte) js.Value {
	array := uint8Array.New(len(data))
	js.CopyBytesToJS(array, data)
	return array
}

func Syscall(call string, params map[string]interface{}) (
	map[string]interface{}, error) {

	params["cmd"] = call

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

	values := map[string]interface{}{
		"ret": result[1].Int(),
	}
	if len(result) > 2 && !result[2].IsUndefined() {
		buf := make([]byte, result[2].Length())
		js.CopyBytesToGo(buf, result[2])
		values["buf"] = buf
	}

	return values, nil
}

func SyscallSetWD(cwd string) {
	syscallSetWD.Invoke(js.ValueOf(cwd))
}
