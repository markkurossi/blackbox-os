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

	"github.com/markkurossi/sandbox-os/kernel/fb"
	"github.com/markkurossi/sandbox-os/kernel/network"
)

var console = fb.NewConsole()

func main() {
	console.Draw()
	log.SetOutput(console)

	log.Printf("Sandbox OS")

	flags := js.PreventDefault | js.StopPropagation
	onKeyboard := js.NewEventCallback(flags, func(event js.Value) {
		evType := event.Get("type").String()
		key := event.Get("key").String()
		keyCode := event.Get("keyCode").Int()
		ctrlKey := event.Get("ctrlKey").Bool()
		log.Printf("%s: key=%s, keyCode=%d, ctrlKey=%v\n",
			evType, key, keyCode, ctrlKey)
	})

	init := js.Global().Get("init")
	init.Invoke(onKeyboard)

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

	<-time.After(5 * time.Second)
	conn.Close()

	if false {
		alert := js.Global().Get("alert")
		alert.Invoke("Hello, Wasm!")
	}
}
