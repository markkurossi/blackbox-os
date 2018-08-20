//
// control.go
//
// Copyright (c) 2018 Markku Rossi
//
// All rights reserved.
//

package control

import (
	"fmt"
)

var (
	KernelPower int    = 1
	WSProxy     string = "localhost:8100"
)

type ValueType int

const (
	String ValueType = iota
	Int
)

type Value struct {
	Name string
	Type ValueType
	Strp *string
	Intp *int
}

func (v Value) String() string {
	switch v.Type {
	case String:
		return fmt.Sprintf("%s=%s", v.Name, *v.Strp)

	case Int:
		return fmt.Sprintf("%s=%d", v.Name, *v.Intp)

	default:
		return fmt.Sprintf("%s=?", v.Name)
	}
}

var Values = []Value{
	Value{
		Name: "kernel.power",
		Type: Int,
		Intp: &KernelPower,
	},
	Value{
		Name: "ws.proxy",
		Type: String,
		Strp: &WSProxy,
	},
}

func Halt() {
	KernelPower = 0
}
