//
// tcp.go
//
// Copyright (c) 2018-2019 Markku Rossi
//
// All rights reserved.
//

package network

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"syscall/js"
	"time"

	"github.com/markkurossi/blackbox-os/lib/encoding"
	"github.com/markkurossi/blackbox-os/lib/wsproxy"
)

var (
	wsNew   = js.Global().Get("webSocketNew")
	wsSend  = js.Global().Get("webSocketSend")
	wsClose = js.Global().Get("webSocketClose")
)

func DialTimeout(proxy, addr string, timeout time.Duration) (net.Conn, error) {
	url := fmt.Sprintf("ws://%s/proxy", proxy)

	conn := &WSConn{
		ws:      NewWebSocket(url),
		network: "tcp",
		addr:    addr,
	}

	// Wait for WebSocket to connect.
	for msg := range conn.ws.C {
		switch msg.Type {
		case Open:
			// Dial.
			req := wsproxy.Dial{
				Addr:    addr,
				Timeout: timeout,
			}
			data, err := encoding.Marshal(&req)
			if err != nil {
				conn.Close()
				return nil, err
			}
			conn.Write(data)

		case Error:
			conn.Close()
			return nil, msg.Error

		case Close:
			return nil, fmt.Errorf("Connection closed")

		case Data:
			status := new(wsproxy.Status)
			err := encoding.Unmarshal(bytes.NewReader(msg.Data), status)
			if err != nil {
				return nil, err
			}
			if !status.Success {
				conn.Close()
				return nil, errors.New(status.Error)
			}
			return conn, nil
		}
	}
	return nil, fmt.Errorf("Connection timeout")
}

type WebSocket struct {
	URL       string
	Native    js.Value
	C         chan Message
	onOpen    js.Func
	onMessage js.Func
	onError   js.Func
	onClose   js.Func
}

func (ws *WebSocket) Network() string {
	return "ws"
}

func (ws *WebSocket) String() string {
	return ws.URL
}

func (ws *WebSocket) Send(data []byte) {
	buf := js.Global().Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(buf, data)
	wsSend.Invoke(ws.Native, buf)
}

func (ws *WebSocket) Close() {
	wsClose.Invoke(ws.Native)

	// Drain message channel
loop:
	for {
		select {
		case <-ws.C:
		default:
			break loop
		}
	}
}

type MessageType int

const (
	Open MessageType = iota
	Error
	Close
	Data
)

type Message struct {
	Type  MessageType
	Error error
	Data  []byte
}

func (m *Message) String() string {
	switch m.Type {
	case Open:
		return "Open"

	case Error:
		return fmt.Sprintf("Error=%s", m.Error)

	case Close:
		return "Close"

	case Data:
		return fmt.Sprintf("Data=%x", m.Data)

	default:
		return fmt.Sprintf("{msg %d}", m.Type)
	}
}

func NewWebSocket(url string) *WebSocket {
	ws := &WebSocket{
		URL: url,
		C:   make(chan Message),
	}
	ws.onOpen = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		ws.C <- Message{
			Type: Open,
		}
		return nil
	})
	ws.onMessage = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) != 1 {
			log.Printf("Invalid onMessage data\n")
			return nil
		}
		data := args[0]

		len := data.Length()
		bytes := make([]byte, len)
		for i := 0; i < len; i++ {
			v := data.Index(i).Int()
			bytes[i] = byte(v)
		}

		ws.C <- Message{
			Type: Data,
			Data: bytes,
		}
		return nil
	})
	ws.onError = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		ws.C <- Message{
			Type:  Error,
			Error: errors.New(args[0].String()),
		}
		return nil
	})
	ws.onClose = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		ws.C <- Message{
			Type: Close,
		}
		return nil
	})

	ws.Native = wsNew.Invoke(url, ws.onOpen, ws.onMessage, ws.onError,
		ws.onClose)

	return ws
}

type WSConn struct {
	ws      *WebSocket
	network string
	addr    string
	data    []byte
}

func (c *WSConn) Read(b []byte) (n int, err error) {
	for len(c.data) == 0 {
	messages:
		for msg := range c.ws.C {
			switch msg.Type {
			case Data:
				c.onData(msg.Data)
				break messages

			case Error:
				return 0, msg.Error

			case Open:
				return 0, fmt.Errorf("Unexpected WebSocket open message")

			case Close:
				return 0, io.EOF
			}
		}
	}

	n = copy(b, c.data)
	c.data = c.data[n:]

	return
}

func (c *WSConn) Write(b []byte) (n int, err error) {
	c.ws.Send(b)
	return len(b), nil
}

func (c *WSConn) Close() error {
	c.ws.Close()
	return nil
}

func (c *WSConn) LocalAddr() net.Addr {
	return c.ws
}

func (c *WSConn) RemoteAddr() net.Addr {
	return c
}

func (c *WSConn) Network() string {
	return c.network
}

func (c *WSConn) String() string {
	return c.addr
}

func (c *WSConn) SetDeadline(t time.Time) error {
	if err := c.SetReadDeadline(t); err != nil {
		return err
	}
	return c.SetWriteDeadline(t)
}

func (c *WSConn) SetReadDeadline(t time.Time) error {
	return fmt.Errorf("SetReadDeadline not implemented yet")
}

func (c *WSConn) SetWriteDeadline(t time.Time) error {
	return fmt.Errorf("SetWriteDeadline not implemented yet")
}

func (c *WSConn) onData(data []byte) {
	c.data = append(c.data, data...)
}
