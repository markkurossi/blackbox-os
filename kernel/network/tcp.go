//
// tcp.go
//
// Copyright (c) 2018 Markku Rossi
//
// All rights reserved.
//

package network

import (
	"fmt"
	"io"
	"net"
	"syscall/js"
	"time"
)

var (
	wsNew   = js.Global().Get("webSocketNew")
	wsSend  = js.Global().Get("webSocketSend")
	wsClose = js.Global().Get("webSocketClose")
)

func DialTimeout(address string, timeout time.Duration) (net.Conn, error) {
	url := "ws://localhost:8100/proxy"

	ws := &WSConn{
		ws: NewWebSocket(url),
	}

	// Wait for WebSocket to connect.
	for msg := range ws.ws.C {
		switch msg.Type {
		case Open:
			return ws, nil

		case Error:
			return nil, msg.Error

		case Close:
			return nil, fmt.Errorf("Connection closed")

		case Data:
			ws.onData(msg.Data)
			return ws, nil
		}
	}
	return nil, fmt.Errorf("Connection timeout")
}

type WebSocket struct {
	URL       string
	Native    js.Value
	C         chan Message
	onOpen    js.Callback
	onMessage js.Callback
	onError   js.Callback
	onClose   js.Callback
}

func (ws *WebSocket) Send(data []byte) {
	buf := make([]byte, len(data))
	copy(buf, data)
	ta := js.TypedArrayOf(buf)

	wsSend.Invoke(ws.Native, ta)

	ta.Release()
}

func (ws *WebSocket) Close() {
	ws.Native.Set("onopen", js.Undefined())
	ws.Native.Set("onerror", js.Undefined())
	ws.Native.Set("onclose", js.Undefined())

	wsClose.Invoke(ws.Native)

	// Drain message channel
loop:
	for {
		select {
		case msg := <-ws.C:
			fmt.Printf("drain: msg %v\n", msg)

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

func NewWebSocket(url string) *WebSocket {
	ws := &WebSocket{
		URL: url,
		C:   make(chan Message),
	}
	flags := js.PreventDefault | js.StopPropagation

	ws.onOpen = js.NewEventCallback(flags, func(event js.Value) {
		fmt.Printf("ws.onOpen\n")
		ws.C <- Message{
			Type: Open,
		}
	})
	ws.onMessage = js.NewCallback(func(args []js.Value) {
		if len(args) != 1 {
			fmt.Printf("Invalid onMessage data\n")
			return
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
	})
	ws.onError = js.NewEventCallback(flags, func(event js.Value) {
		fmt.Printf("ws.onError: %v\n", event)
		ws.C <- Message{
			Type:  Error,
			Error: fmt.Errorf("WS Error: %s", event),
		}
	})
	ws.onClose = js.NewEventCallback(flags, func(event js.Value) {
		fmt.Printf("ws.onClose\n")
		ws.C <- Message{
			Type: Close,
		}
	})

	ws.Native = wsNew.Invoke(url, ws.onMessage)

	ws.Native.Set("onopen", ws.onOpen)
	ws.Native.Set("onerror", ws.onError)
	ws.Native.Set("onclose", ws.onClose)

	return ws
}

type WSConn struct {
	ws   *WebSocket
	data []byte
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
	return c
}

func (c *WSConn) RemoteAddr() net.Addr {
	return c
}

func (c *WSConn) Network() string {
	return "ws"
}

func (c *WSConn) String() string {
	return c.ws.URL
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
