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
	"syscall/js"
	"time"

	"github.com/markkurossi/sandbox-os/kernel/network"
)

func main() {
	init := js.Global().Get("init")
	init.Invoke()

	conn, err := network.DialTimeout("localhost:2252", 5*time.Second)
	if err != nil {
		fmt.Printf("Dial failed: %s\n", err)
		return
	}
	if true {
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

	if false {
		alert := js.Global().Get("alert")
		alert.Invoke("Hello, Wasm!")
	}
}
