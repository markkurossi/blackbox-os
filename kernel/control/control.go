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
	"strconv"
)

var (
	KernelPower int    = 1
	WSProxy     string = "localhost:8100"
	FSRoot      string = fmt.Sprintf("http://%s/fs", WSProxy)
	FSZone      string = "default"
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

func (v *Value) Set(value string) error {
	switch v.Type {
	case String:
		*v.Strp = value
		return nil

	case Int:
		i, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid integer value '%s': %s", value, err)
		}
		*v.Intp = i
		return nil

	default:
		return fmt.Errorf("unknown value type %d", v.Type)
	}
}

func (v *Value) String() string {
	switch v.Type {
	case String:
		return fmt.Sprintf("%s=%s", v.Name, *v.Strp)

	case Int:
		return fmt.Sprintf("%s=%d", v.Name, *v.Intp)

	default:
		return fmt.Sprintf("%s=?", v.Name)
	}
}

var Values = []*Value{
	&Value{
		Name: "kernel.power",
		Type: Int,
		Intp: &KernelPower,
	},
	&Value{
		Name: "ws.proxy",
		Type: String,
		Strp: &WSProxy,
	},
	&Value{
		Name: "fs.root",
		Type: String,
		Strp: &FSRoot,
	},
	&Value{
		Name: "fs.zone",
		Type: String,
		Strp: &FSZone,
	},
}

func Var(name string) (*Value, error) {
	for _, v := range Values {
		if v.Name == name {
			return v, nil
		}
	}
	return nil, fmt.Errorf("unknown oid '%s'", name)
}

func SetVar(name, value string) error {
	for _, v := range Values {
		if v.Name == name {
			return v.Set(value)
		}
	}
	return fmt.Errorf("unknown oid '%s'", name)
}

func Halt() {
	KernelPower = 0
}
