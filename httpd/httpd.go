//
// httpd.go
//
// Copyright (c) 2018 Markku Rossi
//
// All rights reserved.
//
// WebSocket to TCP proxy.

package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/gorilla/websocket"
)

func main() {
	addr := flag.String("addr", "localhost:8100", "HTTP service address")
	directory := flag.String("d", ".", "Directory containing static content")
	flag.Parse()

	http.HandleFunc("/proxy", proxy)
	http.Handle("/", http.FileServer(http.Dir(*directory)))

	log.Printf("Serving %s on HTTP: %s\n", *directory, *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func proxy(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer ws.Close()

	log.Printf("New connection to %s\n", r.URL)

	c, err := net.Dial("tcp", "localhost:2252")
	if err != nil {
		log.Printf("tcp dial: %s\n", err)
		return
	}

	go func() {
		var buf [4096]byte
		for {
			n, err := c.Read(buf[:])
			if err != nil {
				log.Printf("tcp.Read: %s\n", err)
				ws.Close()
				return
			}
			fmt.Printf("TCP->WS:\n%s", hex.Dump(buf[:n]))

			err = ws.WriteMessage(websocket.BinaryMessage, buf[:n])
			if err != nil {
				log.Printf("ws.Write: %s\n", err)
				ws.Close()
				return
			}
		}
	}()

	for {
		_, message, err := ws.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		fmt.Printf("WS->TCP:\n%s", hex.Dump(message))
		_, err = c.Write(message)
		if err != nil {
			log.Printf("tcp.write: %s\n", err)
			break
		}
	}
}
