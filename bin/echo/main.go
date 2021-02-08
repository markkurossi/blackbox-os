//
// Copyright (c) 2021 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"fmt"

	"github.com/markkurossi/blackbox-os/kernel/kmsg"
)

func main() {
	fmt.Fprintf(kmsg.Writer, "Hello, Black Box OS!")
}
