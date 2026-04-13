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
func main() {
	names := []string{"divida-ativa-batch-configmap", "divida-ativa-batch-secret", "divida-ativa-batch-svc", "divida-ativa-batch-deploy", "divida-ativa-batch-route"}
	prefix := ""
	for _, n := range names {
		if prefix == "" { prefix = n } else { prefix = commonPrefix(prefix, n) }
	}
	fmt.Println("Prefix is:", prefix)
}
