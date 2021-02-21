//
// Copyright (c) 2021 Markku Rossi
//
// All rights reserved.
//

package errno

import (
	"errors"
)

var (
	ENOENT = errors.New("ENOENT")
	EINVAL = errors.New("EINVAL")
	ENOSYS = errors.New("ENOSYS")
	EBADF  = errors.New("EBADF")
)
