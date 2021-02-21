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
	"net"
	"os"
	"regexp"
	"time"

	"github.com/markkurossi/blackbox-os/lib/bbos"
	"github.com/markkurossi/blackbox-os/lib/vt100"
	"golang.org/x/crypto/ssh"
)

var reTarget *regexp.Regexp = regexp.MustCompilePOSIX(
	"(([^@]+)@)?([^:]+)(:.*)?")

func main() {
	flag.Parse()

	args := flag.Args()

	if len(args) < 1 {
		fmt.Printf("Usage: ssh [user@]host[:port]\n")
		return
	}

	matches := reTarget.FindStringSubmatch(args[0])
	if matches == nil {
		fmt.Fprintf(os.Stderr, "Invalid target '%s'\n", args[0])
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

	err := sshConnection(user, addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "SSH error: %s\n", err)
	}
}

func sshConnection(user, addr string) error {
	fmt.Printf("Connecting to %s@%s...\n", user, addr)

	conn, err := bbos.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()

	var authMethods = []ssh.AuthMethod{
		ssh.PasswordCallback(func() (secret string, err error) {
			return vt100.ReadPassword(
				fmt.Sprintf("%s@%s's password: ", user, addr))
		}),
	}

	c, chans, reqs, err := ssh.NewClientConn(conn, addr, &ssh.ClientConfig{
		User: user,
		Auth: authMethods,
		HostKeyCallback: func(
			hostname string, remote net.Addr, key ssh.PublicKey) error {
			fmt.Printf("%s key fingerprint is %s.\n",
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
	flags, err := vt100.MakeRaw(os.Stdin)
	if err != nil {
		return err
	}
	defer vt100.MakeCooked(os.Stdin, flags)

	go io.Copy(stdin, os.Stdin)
	go io.Copy(os.Stderr, stderr)

	io.Copy(os.Stdout, stdout)

	return nil
}
