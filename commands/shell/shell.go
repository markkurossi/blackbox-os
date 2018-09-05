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
	"sort"
	"strings"
	"syscall/js"

	"github.com/markkurossi/blackbox-os/kernel/control"
	"github.com/markkurossi/blackbox-os/kernel/process"
	"github.com/markkurossi/blackbox-os/lib/emulator"
	"github.com/markkurossi/blackbox-os/lib/file"
)

type Builtin struct {
	Name string
	Cmd  func(p *process.Process, args []string)
}

var builtin []Builtin

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
	rl.Tab = func(line string) []string {
		return tabCompletion(p, line)
	}

	for control.KernelPower != 0 {
		line, err := rl.Read(prompt(p))
		fmt.Fprintf(p.Stdout, "\n")
		if err != nil {
			fmt.Fprintf(p.Stderr, "%s\n", err)
			return
		}
		args := strings.Split(line, " ")
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

func tabCompletion(p *process.Process, prefix string) []string {
	return []string{"a", "b", "c"}
}
