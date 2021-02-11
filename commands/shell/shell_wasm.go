//
// shell.go
//
// Copyright (c) 2018-2021 Markku Rossi
//
// All rights reserved.
//

package shell

import (
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
	"syscall/js"

	"github.com/markkurossi/backup/lib/tree"
	"github.com/markkurossi/blackbox-os/kernel/control"
	"github.com/markkurossi/blackbox-os/kernel/kmsg"
	"github.com/markkurossi/blackbox-os/kernel/process"
	"github.com/markkurossi/blackbox-os/lib/bbos"
	"github.com/markkurossi/blackbox-os/lib/file"
	"github.com/markkurossi/blackbox-os/lib/vt100"
)

type Builtin struct {
	Name string
	Cmd  func(p *process.Process, args []string)
}

var builtin []Builtin

type CommandLine []string

func (cl CommandLine) String() string {
	var result string

	for idx, command := range cl {
		if idx > 0 {
			result += " "
		}
		result += CommandEscape(command)
	}
	return result
}

func split(line string) CommandLine {
	// XXX proper shell argument splitting
	return strings.Split(line, " ")
}

var reCommandEscape = regexp.MustCompilePOSIX("([ \\'\"])")

func CommandEscape(command string) string {
	return reCommandEscape.ReplaceAllString(command, "\\${1}")
}

func cmd_help(p *process.Process, args []string) {
	fmt.Fprintf(p.Stdout, "Available commands are:\n")

	names := make([]string, 0, len(builtin))
	for _, cmd := range builtin {
		names = append(names, cmd.Name)
	}
	sort.Strings(names)

	for _, name := range names {
		fmt.Fprintf(p.Stdout, "  %s\n", name)
	}
}

func init() {
	builtin = append(builtin, []Builtin{
		Builtin{
			Name: "alert",
			Cmd: func(p *process.Process, args []string) {
				if len(args) < 2 {
					fmt.Fprintf(p.Stdout, "Usage: alert msg\n")
					return
				}
				js.Global().Get("alert").Invoke(strings.Join(args[1:], " "))
			},
		},
		Builtin{
			Name: "halt",
			Cmd: func(p *process.Process, args []string) {
				control.Halt()
			},
		},
		Builtin{
			Name: "help",
			Cmd:  cmd_help,
		},
	}...)
}

func readLine(in io.Reader) string {
	var buf [1024]byte
	var line string

	for {
		n, _ := in.Read(buf[:])
		if n == 0 {
			break
		}
		line += string(buf[:n])
		if buf[n-1] == '\n' || buf[n-1] == '\r' {
			break
		}
	}
	return strings.TrimSpace(line)
}

func Shell(p *process.Process) error {
	rl := vt100.NewReadline(p.TTY, kmsg.Writer)
	rl.Tab = func(line string) (string, []string) {
		return tabCompletion(p, line)
	}

	for control.KernelPower != 0 {
		line, err := rl.Read(prompt(p))
		fmt.Fprintf(p.Stdout, "\n")
		if err != nil {
			return err
		}
		args := split(line)
		if len(args) == 0 || len(args[0]) == 0 {
			continue
		}

		var found bool

		for _, cmd := range builtin {
			if args[0] == cmd.Name {
				found = true
				os.Args = args
				flag.CommandLine = flag.NewFlagSet(args[0],
					flag.ContinueOnError)
				flag.CommandLine.SetOutput(p.Stdout)
				cmd.Cmd(p, args)
				break
			}
		}
		if !found {
			fmt.Fprintf(p.Stderr, "Unknown command '%s'\n", args[0])
		}
	}
	return nil
}

func prompt(p *process.Process) string {
	var result []rune

	prompt := []rune(control.ShellPrompt)

	for i := 0; i < len(prompt); i++ {
		switch prompt[i] {
		case '\\':
			if i+1 < len(prompt) {
				i++
				switch prompt[i] {
				case 'W':
					dir := "{nodir}"

					wd, _, err := p.FS.WD()
					if err == nil {
						parts := file.PathSplit(wd)
						if len(parts) > 0 {
							last := parts[len(parts)-1]
							if len(last) == 0 {
								dir = "/"
							} else {
								dir = last
							}
						}
					}
					result = append(result, []rune(dir)...)

				default:
					result = append(result, prompt[i])
				}
			}

		default:
			result = append(result, prompt[i])
		}
	}

	return string(result)
}

func tabCompletion(p *process.Process, line string) (string, []string) {
	parts := split(line)

	if len(parts) == 0 {
		return line, nil
	}
	last := parts[len(parts)-1]
	if len(last) == 0 {
		return line, nil
	}

	if last[0] == '@' {
		return tabSnapshotCompletion(p, line, parts, last)
	}

	return tabFileCompletion(p, line, parts, last)
}

func tabSnapshotCompletion(p *process.Process, line string, parts CommandLine,
	last string) (string, []string) {

	root := p.FS.Zone().HeadID
	var result []string

	for !root.Undefined() {
		element, err := tree.DeserializeID(root, p.FS.Zone())
		if err != nil {
			return line, nil
		}
		el, ok := element.(*tree.Snapshot)
		if !ok {
			return line, nil
		}
		id := fmt.Sprintf("@%s", root)
		if strings.HasPrefix(id, last) {
			result = append(result, id)
		}
		root = el.Parent
	}

	switch len(result) {
	case 0:
		return line, nil
	case 1:
		parts[len(parts)-1] = result[0]
		return parts.String(), nil
	default:
		parts[len(parts)-1] = commonPrefix(result)
		return parts.String(), result
	}
}

func tabFileCompletion(p *process.Process, line string, parts CommandLine,
	last string) (string, []string) {

	info, err := bbos.Stat(p, last)
	if err == nil {
		// An existing file.
		if info.IsDir() {
			if !strings.HasSuffix(last, "/") {
				parts[len(parts)-1] = fmt.Sprintf("%s/", last)
				return parts.String(), nil
			}
			files, err := bbos.ReadDir(p, last)
			if err != nil {
				return line, nil
			}
			var arr []string
			for _, i := range files {
				name := i.Name()
				if i.IsDir() {
					name += "/"
				}
				arr = append(arr, name)
			}
			switch len(arr) {
			case 0:
				return line, nil
			case 1:
				parts[len(parts)-1] = makeFilename(last, arr[0])
				return parts.String(), nil
			default:
				return parts.String(), arr
			}
		} else {
			// Return the line unmodified.
			return line, nil
		}
	}

	// Check if `last' is a file name prefix.
	path := file.PathSplit(last)
	if len(path) > 0 {
		last = path[len(path)-1]
		path = path[:len(path)-1]
	}
	info, err = bbos.Stat(p, path.String())
	if err != nil {
		return line, nil
	}
	if info.IsDir() {
		files, err := bbos.ReadDir(p, path.String())
		if err != nil {
			return line, nil
		}
		var arr []string
		for _, i := range files {
			if strings.HasPrefix(i.Name(), last) {
				name := i.Name()
				if i.IsDir() {
					name += "/"
				}
				arr = append(arr, name)
			}
		}
		switch len(arr) {
		case 0:
			return line, nil
		case 1:
			parts[len(parts)-1] = makeFilename(path.String(), arr[0])
			return parts.String(), nil
		default:
			parts[len(parts)-1] = commonPrefix(arr)
			return parts.String(), arr
		}
	} else {
		// Return the line unmodified.
		return line, nil
	}
}

func makeFilename(prefix, file string) string {
	if len(prefix) == 0 {
		return file
	} else if prefix[len(prefix)-1] == '/' {
		return fmt.Sprintf("%s%s", prefix, file)
	} else {
		return fmt.Sprintf("%s/%s", prefix, file)
	}
}

func commonPrefix(values []string) string {
	var prefix string

	for idx, val := range values {
		if idx == 0 {
			prefix = val
		}
		var l = len(prefix)
		if len(val) < l {
			l = len(val)
		}
		var i int
		for i = 0; i < l; i++ {
			if prefix[i] != val[i] {
				break
			}
		}
		prefix = prefix[:i]
	}
	return prefix
}
