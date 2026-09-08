package httpingress

// ParseLimits caps hostile HTTP/1 wire patterns on the tracker ingress path.
// Zero MinChunkedDataBytes disables the micro-chunk floor.
type ParseLimits struct {
	MinChunkedDataBytes int
}
