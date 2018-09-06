//
// shell.go
//
// Copyright (c) 2018 Markku Rossi
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

	"github.com/markkurossi/blackbox-os/kernel/control"
	"github.com/markkurossi/blackbox-os/kernel/process"
	"github.com/markkurossi/blackbox-os/lib/bbos"
	"github.com/markkurossi/blackbox-os/lib/emulator"
	"github.com/markkurossi/blackbox-os/lib/file"
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
		if buf[n-1] == '\n' {
			break
		}
	}
	return strings.TrimSpace(line)
}

func Shell(p *process.Process) {
	rl := emulator.NewReadline(p.TTY)
	rl.Tab = func(line string) (string, []string) {
		return tabCompletion(p, line)
	}

	for control.KernelPower != 0 {
		line, err := rl.Read(prompt(p))
		fmt.Fprintf(p.Stdout, "\n")
		if err != nil {
			fmt.Fprintf(p.Stderr, "%s\n", err)
			return
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

					wd, _, err := p.WD()
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

	// XXX Check what to complete

	return tabFileCompletion(p, line, parts, last)
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
				parts[len(parts)-1] = fmt.Sprintf("%s%s", last, arr[0])
				return parts.String(), nil
			default:
				return "", arr
			}
		} else {
			// Return the line unmodified.
			return line, nil
		}
	}

	// Check if `last' is a file name prefix.
	lastPath := file.PathSplit(last)
	if len(lastPath) > 0 {
		last = lastPath[len(lastPath)-1]
		lastPath = lastPath[:len(lastPath)-1]
	}
	info, err = bbos.Stat(p, lastPath.String())
	if err != nil {
		return line, nil
	}
	if info.IsDir() {
		files, err := bbos.ReadDir(p, lastPath.String())
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
			parts[len(parts)-1] = fmt.Sprintf("%s%s", lastPath.String(), arr[0])
			return parts.String(), nil
		default:
			return "", arr
		}
	} else {
		// Return the line unmodified.
		return line, nil
	}
}
