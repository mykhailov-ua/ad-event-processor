package gtax

// ComputeMicro applies basis-point gross receipts tax to a CTV spend subtotal.
func ComputeMicro(spendMicro int64, rateBPS int32) int64 {
	if spendMicro <= 0 || rateBPS <= 0 {
		return 0
	}
	return (spendMicro*int64(rateBPS) + 5000) / 10000
}
