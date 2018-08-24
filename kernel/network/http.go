//
// http.go
//
// Copyright (c) 2018 Markku Rossi
//
// All rights reserved.
//

package network

import (
	"syscall/js"
)

var (
	httpGet = js.Global().Get("httpGet")
)

func HttpGet(url string) {
	httpGet.Invoke(url)
}
