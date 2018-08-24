//
// cmd_http.go
//
// Copyright (c) 2018 Markku Rossi
//
// All rights reserved.
//

package shell

import (
	"fmt"
	"net/http"

	"github.com/markkurossi/blackbox-os/kernel/process"
)

func init() {
	builtin = append(builtin, Builtin{
		Name: "http",
		Cmd:  cmd_http,
	})
}

func cmd_http(p *process.Process, args []string) {
	client := http.Client{}
	req, err := http.NewRequest("GET", "https://golang.org/", nil)
	if err != nil {
		fmt.Printf("HTTP error: %s\n", err)
		return
	}
	req.Header.Add("js.fetch:mode", "no-cors")

	fmt.Println("debug4")
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("debug5")
		fmt.Printf("HTTP error: %s\n", err)
		return
	}
	fmt.Println("debug6")
	resp.Body.Close()
}
