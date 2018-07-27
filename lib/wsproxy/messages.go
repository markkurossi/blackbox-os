//
// messages.go
//
// Copyright (c) 2018 Markku Rossi
//
// All rights reserved.
//

package wsproxy

import (
	"time"
)

type Dial struct {
	Addr    string
	Timeout time.Duration
}

type Status struct {
	Success bool
	Error   string
}
