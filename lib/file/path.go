//
// path.go
//
// Copyright (c) 2018 Markku Rossi
//
// All rights reserved.
//

package file

import (
	"regexp"
)

var rePathEscape = regexp.MustCompilePOSIX("([/])")

type Path []string

func (p Path) String() string {
	var result string

	if len(p) == 1 && len(p[0]) == 0 {
		return "/"
	}

	for idx, el := range p {
		if idx > 0 {
			result += "/"
		}
		result += PathEscape(el)
	}
	return result
}

func PathEscape(path string) string {
	return rePathEscape.ReplaceAllString(path, "\\${1}")
}

func PathSplit(path string) Path {
	var result []string
	var runes = []rune(path)
	var part []rune

	for i := 0; i < len(runes); i++ {
		if runes[i] == '/' {
			result = append(result, string(part))
			part = nil
		} else if runes[i] == '\\' && i+1 < len(runes) {
			i++
			part = append(part, runes[i])
		} else {
			part = append(part, runes[i])
		}
	}
	if len(part) > 0 {
		result = append(result, string(part))
	}
	return result
}
