//
// cmd_date.go
//
// Copyright (c) 2019-2021 Markku Rossi
//
// All rights reserved.
//

package shell

import (
	"fmt"
	"time"
)

func init() {
	builtin = append(builtin, Builtin{
		Name: "date",
		Cmd:  cmd_date,
	})
}

func cmd_date(args []string) {
	now := time.Now()
	fmt.Printf("%s\n", now.Format(time.UnixDate))
}
