package opkey

var (
	depthFn func(float64)
	shedFn  func()
)

// BindMetrics wires OpKeyPool gauges and shed counter to Prometheus callbacks.
func BindMetrics(depth func(float64), shed func()) {
	depthFn = depth
	shedFn = shed
}

func setDepth(v float64) {
	if fn := depthFn; fn != nil {
		fn(v)
	}
}

func incIngressShed() {
	if fn := shedFn; fn != nil {
		fn()
	}
}
