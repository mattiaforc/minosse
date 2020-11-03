package main

import "strings"

func hashmapMapToString(m map[string]string, f func(string, string) string) string {
	var str strings.Builder

	for k, v := range m {
		str.WriteString(f(k, v))
	}
	return str.String()
}
