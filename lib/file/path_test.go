//
// path_test.go
//
// Copyright (c) 2018 Markku Rossi
//
// All rights reserved.
//

package file

import (
	"reflect"
	"testing"
)

var pathTests = map[string]string{
	"a/b": "a\\/b",
}

func TestPathEscape(t *testing.T) {
	for from, to := range pathTests {
		str := PathEscape(from)
		if str != to {
			t.Errorf("Path escape failed: %s=>%s, expected %s\n", from, str, to)
		}
	}
}

var splitTests = map[string][]string{
	"a":      []string{"a"},
	"a\\/b":  []string{"a/b"},
	"/a/b/c": []string{"", "a", "b", "c"},
}

func TestPathSplit(t *testing.T) {
	for from, to := range splitTests {
		arr := PathSplit(from)

		if !reflect.DeepEqual(to, arr) {
			t.Errorf("Path split failed: %s=>%v, expected %v\n", from, arr, to)
		}
	}
}
