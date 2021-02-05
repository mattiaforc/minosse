package main

import (
	"fmt"
	"strings"
)

func hashmapMapToString(m map[string]string, f func(string, string) string) string {
	var str strings.Builder

	for k, v := range m {
		str.WriteString(f(k, v))
	}
	return str.String()
}

func printMinosse() {
	asciiArt :=
		`
 __   __  ___   __    _  _______  _______  _______  _______ 
|  |_|  ||   | |  |  | ||       ||       ||       ||       |
|       ||   | |   |_| ||   _   ||  _____||  _____||    ___|
|       ||   | |       ||  | |  || |_____ | |_____ |   |___ 
|       ||   | |  _    ||  |_|  ||_____  ||_____  ||    ___|
| ||_|| ||   | | | |   ||       | _____| | _____| ||   |___ 
|_|   |_||___| |_|  |__||_______||_______||_______||_______|


`
	fmt.Println(asciiArt)
}
