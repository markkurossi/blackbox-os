//
// main.go
//
// Copyright (c) 2018-2021 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/markkurossi/blackbox-os/lib/bbos"
	"github.com/markkurossi/blackbox-os/lib/file"
	"github.com/markkurossi/blackbox-os/lib/readline"
)

var shellPrompt = "bbos \\W $ "

type Builtin struct {
	Name string
	Cmd  func(args []string)
}

var (
	builtin  []Builtin
	builtins map[string]Builtin
	running  = true
)

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

func cmd_help(args []string) {
	fmt.Fprintf(os.Stdout, "Available commands are:\n")

	names := make([]string, 0, len(builtin))
	for _, cmd := range builtin {
		names = append(names, cmd.Name)
	}
	sort.Strings(names)

	for _, name := range names {
		fmt.Fprintf(os.Stdout, "  %s\n", name)
	}
}

func init() {
	builtin = append(builtin, []Builtin{
		// Builtin{
		// 	Name: "alert",
		// 	Cmd: func(p *process.Process, args []string) {
		// 		if len(args) < 2 {
		// 			fmt.Fprintf(p.Stdout, "Usage: alert msg\n")
		// 			return
		// 		}
		// 		js.Global().Get("alert").Invoke(strings.Join(args[1:], " "))
		// 	},
		// },
		Builtin{
			Name: "exit",
			Cmd: func(args []string) {
				running = false
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

func main() {
	builtins = make(map[string]Builtin)
	for _, bi := range builtin {
		builtins[bi.Name] = bi
	}

	rl := readline.NewReadline(os.Stdin, os.Stdout, os.Stderr)
	rl.Tab = func(line string) (string, []string) {
		return tabCompletion(line)
	}

	for running {
		line, err := rl.Read(prompt())
		fmt.Fprintf(os.Stdout, "\n")
		if err != nil {
			log.Fatal(err)
		}
		args := split(line)
		if len(args) == 0 || len(args[0]) == 0 {
			continue
		}

		err = runCommand(args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %s\n", args[0], err)
		}
	}
}

func runCommand(args []string) error {
	bi, ok := builtins[args[0]]
	if ok {
		os.Args = args
		flag.CommandLine = flag.NewFlagSet(args[0], flag.ContinueOnError)
		flag.CommandLine.SetOutput(os.Stdout)
		bi.Cmd(args)
	} else {
		// Run as process.
		pid, err := bbos.Spawn(args, []int{
			int(os.Stdin.Fd()),
			int(os.Stdout.Fd()),
			int(os.Stderr.Fd()),
		})
		if err != nil {
			return err
		}
		code, err := bbos.Wait(pid)
		if err != nil {
			return err
		}
		if code != 0 {
			fmt.Printf("%d: Exit %d: %s\n", pid, code, args[0])
		}
	}
	return nil
}

func prompt() string {
	var result []rune

	prompt := []rune(shellPrompt)

	for i := 0; i < len(prompt); i++ {
		switch prompt[i] {
		case '\\':
			if i+1 < len(prompt) {
				i++
				switch prompt[i] {
				case 'W':
					dir := "{nodir}"

					wd, err := os.Getwd()
					if err != nil {
						fmt.Fprintf(os.Stderr, "Getwd: %s\n", err)
					}
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

func tabCompletion(line string) (string, []string) {
	parts := split(line)

	if len(parts) == 0 {
		return line, nil
	}
	last := parts[len(parts)-1]
	if len(last) == 0 {
		return line, nil
	}

	// 	if last[0] == '@' {
	// 		return tabSnapshotCompletion(p, line, parts, last)
	// 	}
	//
	return tabFileCompletion(line, parts, last)
}

// func tabSnapshotCompletion(p *process.Process, line string, parts CommandLine,
// 	last string) (string, []string) {
//
// 	root := p.FS.Zone().HeadID
// 	var result []string
//
// 	for !root.Undefined() {
// 		element, err := tree.DeserializeID(root, p.FS.Zone())
// 		if err != nil {
// 			return line, nil
// 		}
// 		el, ok := element.(*tree.Snapshot)
// 		if !ok {
// 			return line, nil
// 		}
// 		id := fmt.Sprintf("@%s", root)
// 		if strings.HasPrefix(id, last) {
// 			result = append(result, id)
// 		}
// 		root = el.Parent
// 	}
//
// 	switch len(result) {
// 	case 0:
// 		return line, nil
// 	case 1:
// 		parts[len(parts)-1] = result[0]
// 		return parts.String(), nil
// 	default:
// 		parts[len(parts)-1] = commonPrefix(result)
// 		return parts.String(), result
// 	}
// }

func tabFileCompletion(line string, parts CommandLine, last string) (
	string, []string) {

	info, err := os.Stat(last)
	if err == nil {
		// An existing file.
		if info.IsDir() {
			if !strings.HasSuffix(last, "/") {
				parts[len(parts)-1] = fmt.Sprintf("%s/", last)
				return parts.String(), nil
			}
			files, err := ioutil.ReadDir(last)
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
	var dir string
	if len(path) > 1 {
		last = path[len(path)-1]
		path = path[:len(path)-1]
		dir = path.String()
	} else {
		path = []string{}
		dir = "."
	}
	info, err = os.Stat(dir)
	if err != nil {
		return line, nil
	}
	if info.IsDir() {
		files, err := ioutil.ReadDir(dir)
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
