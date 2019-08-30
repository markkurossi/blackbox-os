//
// cmd_ssh.go
//
// Copyright (c) 2018 Markku Rossi
//
// All rights reserved.
//

package shell

import (
	"fmt"
	"io"
	"net"
	"regexp"
	"time"

	"github.com/markkurossi/blackbox-os/kernel/control"
	"github.com/markkurossi/blackbox-os/kernel/network"
	"github.com/markkurossi/blackbox-os/kernel/process"
	"github.com/markkurossi/blackbox-os/lib/emulator"
	"golang.org/x/crypto/ssh"
)

func init() {
	builtin = append(builtin, Builtin{
		Name: "ssh",
		Cmd:  cmd_ssh,
	})
}

var reTarget *regexp.Regexp = regexp.MustCompilePOSIX(
	"(([^@]+)@)?([^:]+)(:.*)?")

func cmd_ssh(p *process.Process, args []string) {
	if len(args) < 2 {
		fmt.Fprintf(p.Stdout, "Usage: ssh [user@]host[:port]\n")
		return
	}

	matches := reTarget.FindStringSubmatch(args[1])
	if matches == nil {
		fmt.Fprintf(p.Stderr, "Invalid target '%s'\n", args[1])
		return
	}

	user := matches[2]
	host := matches[3]
	port := matches[4]

	if len(user) == 0 {
		user = "mtr"
	}

	if len(port) == 0 {
		port = ":22"
	}
	addr := host + port

	err := sshConnection(p, user, addr)
	if err != nil {
		fmt.Fprintf(p.Stderr, "SSH error: %s\n", err)
	}
}

func sshConnection(p *process.Process, user, addr string) error {
	fmt.Fprintf(p.Stdout, "Connecting to %s@%s...\n", user, addr)

	conn, err := network.DialTimeout(control.WSProxy, addr, 5*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()

	var authMethods = []ssh.AuthMethod{
		ssh.PasswordCallback(func() (secret string, err error) {
			fmt.Fprintf(p.Stdout, "%s@%s's password: ", user, addr)
			flags := p.TTY.Flags()
			p.TTY.SetFlags(flags & ^emulator.ECHO)
			defer p.TTY.SetFlags(flags)
			passwd := readLine(p.Stdin)
			return passwd, nil
		}),
	}

	c, chans, reqs, err := ssh.NewClientConn(conn, addr, &ssh.ClientConfig{
		User: user,
		Auth: authMethods,
		HostKeyCallback: func(
			hostname string, remote net.Addr, key ssh.PublicKey) error {
			fmt.Fprintf(p.Stdout, "%s key fingerprint is %s.\n",
				key.Type(), ssh.FingerprintSHA256(key))
			return nil
		},
		Timeout: 5 * time.Minute,
	})
	if err != nil {
		return err
	}

	client := ssh.NewClient(c, chans, reqs)

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	err = session.Setenv("LANG", "en_US.UTF-8")
	if err != nil {
		return err
	}
	err = session.RequestPty("xterm", 24, 80, ssh.TerminalModes{})
	if err != nil {
		return err
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		return err
	}

	err = session.Shell()
	if err != nil {
		return err
	}

	// Enable raw mode for input.
	flags := p.TTY.Flags()
	p.TTY.SetFlags(flags & ^(emulator.ICANON | emulator.ECHO))
	defer p.TTY.SetFlags(flags)

	go io.Copy(stdin, p.Stdin)
	go io.Copy(p.Stderr, stderr)

	io.Copy(p.Stdout, stdout)

	return nil
}
