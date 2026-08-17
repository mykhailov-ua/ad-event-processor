// Package gtax computes tax amounts from spend micro-units and basis points.
package gtax

func ComputeMicro(spendMicro int64, rateBPS int32) int64 {
	if spendMicro <= 0 || rateBPS <= 0 {
		return 0
	}
	return (spendMicro*int64(rateBPS) + 5000) / 10000
}
