package faultproof

import (
	"fmt"
	"strings"
)

// Print writes a fault_proof line to stdout for lab scripts and CLIs.
func Print(fault string, kv map[string]string) {
	var b strings.Builder
	b.WriteString("fault_proof fault=")
	b.WriteString(fault)
	for k, v := range kv {
		b.WriteByte(' ')
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(v)
	}
	fmt.Println(b.String())
}
