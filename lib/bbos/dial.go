//
// Copyright (c) 2021 Markku Rossi
//
// All rights reserved.
//

package bbos

import (
	"fmt"
	"net"
	"time"
)

var (
	_ net.Conn = &Conn{}
)

func DialTimeout(network, address string, timeout time.Duration) (
	net.Conn, error) {

	data, err := Syscall("dial", map[string]interface{}{
		"network": network,
		"address": address,
		"timeout": int64(timeout),
	})
	if err != nil {
		return nil, err
	}
	fd, ok := data["ret"]
	if !ok {
		return nil, fmt.Errorf("DialTimeout: invalid response")
	}
	ifd, ok := fd.(int)
	if !ok {
		return nil, fmt.Errorf("DialTimeout: invalid response")
	}

	return &Conn{
		fd: ifd,
		local: &Addr{
			network: network,
		},
		remote: &Addr{
			network: network,
			address: address,
		},
	}, nil
}

type Addr struct {
	network string
	address string
}

func (a *Addr) Network() string {
	return a.network
}

func (a *Addr) String() string {
	return a.address
}

type Conn struct {
	fd     int
	local  *Addr
	remote *Addr
}

func (c *Conn) Read(b []byte) (n int, err error) {
	return Read(c.fd, b)
}

func (c *Conn) Write(b []byte) (n int, err error) {
	return Write(c.fd, b)
}

func (c *Conn) Close() error {
	return fmt.Errorf("conn.Close: not implemented yet")
}

func (c *Conn) LocalAddr() net.Addr {
	return c.local
}

func (c *Conn) RemoteAddr() net.Addr {
	return c.remote
}

func (c *Conn) SetDeadline(t time.Time) error {
	if err := c.SetReadDeadline(t); err != nil {
		return err
	}
	return c.SetWriteDeadline(t)
}

func (c *Conn) SetReadDeadline(t time.Time) error {
	return fmt.Errorf("SetReadDeadline not implemented yet")
}

func (c *Conn) SetWriteDeadline(t time.Time) error {
	return fmt.Errorf("SetWriteDeadline not implemented yet")
}
