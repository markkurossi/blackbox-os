//
// kernel.go
//
// Copyright (c) 2018-2019 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"fmt"
	"net/url"
	"syscall/js"

	"github.com/markkurossi/blackbox-os/kernel/control"
)

var (
	locationURL = js.Global().Get("location").Get("href").String()
)

func parseParams() {
	url, err := url.Parse(locationURL)
	if err != nil {
		fmt.Fprintf(console, "Failed to parse location URL '%s': %s\n",
			locationURL, err)
	}
	url.RawQuery = ""
	url.Fragment = ""

	control.FSRoot = fmt.Sprintf("%sfs", url)
}
