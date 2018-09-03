//
// cmd_filesystem.go
//
// Copyright (c) 2018 Markku Rossi
//
// All rights reserved.
//

package shell

import (
	"flag"
	"fmt"
	"time"

	"github.com/markkurossi/backup/lib/tree"
	"github.com/markkurossi/blackbox-os/kernel/process"
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

func cmd_pwd(p *process.Process, args []string) {
	str, _, err := p.WD()
	if err != nil {
		fmt.Fprintf(p.Stderr, "pwd: %s\n", err)
	} else {
		fmt.Fprintf(p.Stdout, "%s\n", str)
	}
}

func cmd_cd(p *process.Process, args []string) {
	var err error
	if len(args) < 2 {
		err = p.SetWD("/")
	} else {
		err = p.SetWD(args[1])
	}
	if err != nil {
		fmt.Fprintf(p.Stderr, "chdir: %s\n", err)
	}
}

func cmd_ls(p *process.Process, args []string) {
	long := flag.Bool("l", false, "List in long format.")
	snapshot := flag.Bool("s", false, "List snapshots.")
	flag.Parse()

	if *snapshot {
		err := listSnapshots(p)
		if err != nil {
			fmt.Fprintf(p.Stderr, "ls: %s\n", err)
		}
		return
	}

	_, id, err := p.WD()
	if err != nil {
		fmt.Fprintf(p.Stderr, "ls: %s\n", err)
		return
	}
	element, err := tree.DeserializeID(id, p.FS.Zone)
	if err != nil {
		fmt.Fprintf(p.Stderr, "ls: %s\n", err)
		return
	}

	switch el := element.(type) {
	case *tree.Directory:
		if *long {
			listDirLong(p, el)
		} else {
			listDirShort(p, el)
		}

	default:
		fmt.Fprintf(p.Stderr, "Invalid working directory: %T\n", el)
	}
}

func listSnapshots(p *process.Process) error {
	root := p.FS.Zone.HeadID

	for !root.Undefined() {
		element, err := tree.DeserializeID(root, p.FS.Zone)
		if err != nil {
			return err
		}
		el, ok := element.(*tree.Snapshot)
		if !ok {
			return fmt.Errorf("Invalid snapshot element: %T\n", element)
		}
		selected := " "
		if el.Root.Equal(p.FS.WD[0].ID) {
			selected = "*"
		}
		fmt.Fprintf(p.Stdout, "%s%s\t%s\n", selected, el.Root,
			time.Unix(0, el.Timestamp))
		root = el.Parent
	}
	return nil
}

func listDirShort(p *process.Process, el *tree.Directory) {
	var max = 0

	// Count the length of the longest element.
	for _, e := range el.Entries {
		len := len(e.Name)
		if len > max {
			max = len
		}
	}

	width := (max/8 + 1) * 8
	perLine := 80 / width
	if perLine < 1 {
		perLine = 1
	}

	count := 0

	for _, e := range el.Entries {
		fmt.Fprintf(p.Stdout, "%s", e.Name)
		count++
		if count >= perLine {
			fmt.Fprintf(p.Stdout, "\n")
			count = 0
		} else {
			len := len(e.Name)
			len = (len/8 + 1) * 8
			fmt.Fprintf(p.Stdout, "\t")

			for len < width {
				fmt.Fprintf(p.Stdout, "\t")
				len += 8
			}
		}
	}
	if count > 0 {
		fmt.Fprintf(p.Stdout, "\n")
	}
}

func listDirLong(p *process.Process, el *tree.Directory) {
	now := time.Now()
	for _, e := range el.Entries {
		modified := time.Unix(e.ModTime, 0)
		var modStr string
		if modified.Year() != now.Year() {
			modStr = modified.Format("Jan _2  2006")
		} else {
			modStr = modified.Format("Jan _2 15:04")
		}
		fmt.Fprintf(p.Stdout, "%s  %s\t%s\n", e.Mode, modStr, e.Name)
	}
}
