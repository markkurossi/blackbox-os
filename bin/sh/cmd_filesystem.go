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
	"io"
	"io/ioutil"
	"os"

	"github.com/markkurossi/blackbox-os/lib/bbos"
	"github.com/markkurossi/blackbox-os/lib/vt100"
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
		Builtin{
			Name: "cat",
			Cmd:  cmd_cat,
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
	args = args[1:]
	switch len(args) {
	case 0:
		ls(".")

	case 1:
		ls(args[0])

	default:
		for idx, arg := range args {
			if idx > 0 {
				fmt.Println()
			}
			fmt.Printf("%s:\n", arg)
			ls(arg)
		}
	}
}

func ls(dir string) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ls: %s\n", err)
		return
	}
	var names []string
	for _, f := range files {
		names = append(names, f.Name())
	}
	vt100.Tabulate(names, os.Stdout)
}

func cmd_cat(args []string) {
	for i := 1; i < len(args); i++ {
		file, err := os.Open(args[i])
		if err != nil {
			fmt.Fprintf(os.Stderr, "cat: %s: %s\n", args[i], err)
			continue
		}
		defer file.Close()

		_, err = io.Copy(os.Stdout, file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cat: %s: %s\n", args[i], err)
		}
	}
}
