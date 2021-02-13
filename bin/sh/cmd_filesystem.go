//
// cmd_filesystem.go
//
// Copyright (c) 2018-2021 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/markkurossi/blackbox-os/lib/bbos"
)

func init() {
	builtin = append(builtin, []Builtin{
		Builtin{
			Name: "pwd",
			Cmd:  cmd_pwd,
		},
		Builtin{
			Name: "cd",
			Cmd:  cmd_cd,
		},
		Builtin{
			Name: "ls",
			Cmd:  cmd_ls,
		},
	}...)
}

func cmd_pwd(args []string) {
	str, err := bbos.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "pwd: %s\n", err)
	} else {
		fmt.Printf("%s\n", str)
	}
}

func cmd_cd(args []string) {
	var err error
	if len(args) < 2 {
		err = bbos.Chdir("/")
	} else {
		err = bbos.Chdir(args[1])
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "cd: %s\n", err)
	}
}

func cmd_ls(args []string) {
	files, err := ioutil.ReadDir("/")
	if err != nil {
		fmt.Fprintf(os.Stderr, "ls: %s\n", err)
		return
	}
	fmt.Printf("%v\n", files)
}
