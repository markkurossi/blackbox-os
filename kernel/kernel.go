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
	"log"
	"syscall/js"
	"time"

	"github.com/markkurossi/blackbox-os/kernel/control"
	"github.com/markkurossi/blackbox-os/kernel/network"
	"github.com/markkurossi/blackbox-os/kernel/tty"
)

var console = tty.NewConsole()

func main() {
	console.Flush()
	log.SetOutput(console)

	log.Printf("Black Box OS")

	conn, err := network.DialTimeout("localhost:2252", 5*time.Second)
	if err != nil {
		log.Printf("Dial failed: %s\n", err)
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
				log.Printf("conn:\n%s", hex.Dump(buf[:n]))
			}
		}()
	}

	for control.HasPower {
		<-time.After(5 * time.Second)
	}
	log.Printf("powering down\n")
	conn.Close()

	if false {
		alert := js.Global().Get("alert")
		alert.Invoke("Hello, Wasm!")
	}
}
