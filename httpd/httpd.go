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
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/markkurossi/sandbox-os/lib/encoding"
	"github.com/markkurossi/sandbox-os/lib/wsproxy"
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
		log.Printf("WebSocket upgrade failed: %s\n", err)
		return
	}
	defer ws.Close()

	_, msg, err := ws.ReadMessage()
	if err != nil {
		sendStatus(ws, false,
			fmt.Sprintf("Failed to read dial message: %s", err))
		return
	}
	dial := new(wsproxy.Dial)
	err = encoding.Unmarshal(bytes.NewReader(msg), dial)
	if err != nil {
		sendStatus(ws, false, fmt.Sprintf("Invalid dial message: %s", err))
		return
	}

	log.Printf("New connection to %s\n", dial.Addr)

	c, err := net.DialTimeout("tcp", dial.Addr, dial.Timeout)
	if err != nil {
		sendStatus(ws, false, err.Error())
		return
	}
	err = sendStatus(ws, true, "")
	if err != nil {
		log.Printf("Failed to send connect message: %s\n", err)
		return
	}

	go func() {
		var buf [4096]byte
		for {
			n, err := c.Read(buf[:])
			if err != nil {
				log.Printf("TCP read failed: %s\n", err)
				ws.Close()
				return
			}
			fmt.Printf("TCP->WS:\n%s", hex.Dump(buf[:n]))

			err = ws.WriteMessage(websocket.BinaryMessage, buf[:n])
			if err != nil {
				log.Printf("WebSocket write failed: %s\n", err)
				ws.Close()
				return
			}
		}
	}()

	for {
		_, message, err := ws.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read failed: %s\n", err)
			break
		}
		fmt.Printf("WS->TCP:\n%s", hex.Dump(message))
		_, err = c.Write(message)
		if err != nil {
			log.Printf("TCP write failed: %s\n", err)
			break
		}
	}
}

func sendStatus(ws *websocket.Conn, success bool, msg string) error {
	log.Printf("Status: success=%v, msg=%s\n", success, msg)
	data, err := encoding.Marshal(&wsproxy.Status{
		Success: success,
		Error:   msg,
	})
	if err != nil {
		return err
	}
	return ws.WriteMessage(websocket.BinaryMessage, data)
}
