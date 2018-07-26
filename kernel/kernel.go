//
// kernel.go
//
// Copyright (c) 2018 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"encoding/hex"
	"fmt"
	"io"
	"syscall/js"
	"time"

	"github.com/markkurossi/sandbox-os/kernel/network"
)

var stdin io.WriteCloser

func makeByteSlice(data js.Value) []byte {
	len := data.Length()
	bytes := make([]byte, len)
	for i := 0; i < len; i++ {
		v := data.Index(i).Int()
		bytes[i] = byte(v)
	}
	return bytes
}

func main() {
	init := js.Global().Get("init")
	init.Invoke()

	conn, err := network.DialTimeout("localhost:2252", 5*time.Second)
	if err != nil {
		fmt.Printf("Dial failed: %s\n", err)
		return
	}
	if false {
		go func() {
			var buf [1024]byte
			for {
				n, err := conn.Read(buf[:])
				if err != nil {
					return
				}
				fmt.Printf("conn:\n%s", hex.Dump(buf[:n]))
			}
		}()
	}

	<-time.After(5 * time.Second)
	conn.Close()

	alert := js.Global().Get("alert")
	alert.Invoke("Hello, Wasm!")
}
