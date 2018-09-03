//
// cmd_http.go
//
// Copyright (c) 2018 Markku Rossi
//
// All rights reserved.
//

package shell

import (
	"flag"
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
	method := flag.String("m", "GET", "HTTP method to use (default is GET).")
	flag.Parse()

	if len(flag.Args()) == 0 {
		fmt.Fprintf(p.Stderr, "usage: %s OPTIONS... URL\n")
		return
	}

	client := http.Client{}
	for _, url := range flag.Args() {
		req, err := http.NewRequest(*method, url, nil)
		if err != nil {
			fmt.Fprintf(p.Stderr, "%s %s: %s\n", *method, url, err)
			return
		}
		// req.Header.Add("js.fetch:mode", "no-cors")

		resp, err := client.Do(req)
		if err != nil {
			fmt.Fprintf(p.Stderr, "%s %s: %s\n", *method, url, err)
			return
		}
		resp.Body.Close()
	}
}
