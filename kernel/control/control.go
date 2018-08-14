//
// control.go
//
// Copyright (c) 2018 Markku Rossi
//
// All rights reserved.
//

package control

var HasPower bool = true

func Halt() {
	HasPower = false
}
