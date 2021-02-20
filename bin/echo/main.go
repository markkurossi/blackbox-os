//
// Copyright (c) 2021 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"flag"
	"fmt"
	"strings"
)

func main() {
	suppressNewline := flag.Bool("n", false, "suppress trailing newline")
	escapes := flag.Bool("e", false, "interpret backslash escapes")
	flag.Parse()

	var sb strings.Builder

	for idx, arg := range flag.Args() {
		if idx > 0 {
			sb.WriteRune(' ')
		}
		if *escapes {
			runes := []rune(arg)
			for i := 0; i < len(runes); i++ {
				switch runes[i] {
				case '\\':
					if i+1 >= len(runes) {
						sb.WriteRune(runes[i])
						continue
					}
					i++
					switch runes[i] {
					case 'a':
						sb.WriteRune('\a')
					case 'b':
						sb.WriteRune('\b')
					case 'c':
						*suppressNewline = true
					case 'E':
						sb.WriteRune('\033')
					case 'f':
						sb.WriteRune('\f')
					case 'n':
						sb.WriteRune('\n')
					case 'r':
						sb.WriteRune('\r')
					case 't':
						sb.WriteRune('\t')
					case 'v':
						sb.WriteRune('\v')
					case '\\':
						sb.WriteRune('\\')
					case '0':
						var val int
						for j := 0; j < 3; j++ {
							if i+1 >= len(runes) || runes[i+1] < '0' ||
								runes[i+1] > '7' {
								break
							}
							i++
							val *= 8
							val += int(runes[i] - '0')
						}
						sb.WriteRune(rune(val))

					default:
						sb.WriteRune('\\')
						sb.WriteRune(runes[i])
					}

				default:
					sb.WriteRune(runes[i])
				}
			}
		} else {
			sb.WriteString(arg)
		}
	}
	if *suppressNewline {
		fmt.Print(sb.String())
	} else {
		fmt.Println(sb.String())
	}
}
