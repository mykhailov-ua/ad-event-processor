// Package faultproof logs structured fault-injection proof lines for resilience drills.
package faultproof

import (
	"strings"
	"testing"
)

func Log(t testing.TB, fault string, kv map[string]string) {
	t.Helper()
	var b strings.Builder
	b.WriteString("fault_proof fault=")
	b.WriteString(fault)
	for k, v := range kv {
		b.WriteByte(' ')
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(v)
	}
	t.Log(b.String())
}
