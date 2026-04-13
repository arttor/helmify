package main

import (
	"fmt"
)

func commonPrefix(one, two string) string {
	runes1 := []rune(one)
	runes2 := []rune(two)
	min := len(runes1)
	if min > len(runes2) {
		min = len(runes2)
	}
	for i := 0; i < min; i++ {
		if runes1[i] != runes2[i] {
			return string(runes1[:i])
		}
	}
	return string(runes1[:min])
}

func detect(names []string) string {
	p := ""
	for _, n := range names {
		if p == "" {
			p = n
		} else {
			p = commonPrefix(p, n)
		}
	}
	return p
}

func main() {
	names := []string{
		"divida-ativa-batch-deploy",
		"divida-ativa-batch-svc",
		"divida-ativa-batch-route",
		"divida-ativa-batch-configmap",
		"divida-ativa-batch-secret",
	}
	fmt.Println("Common Prefix:", detect(names))
}
