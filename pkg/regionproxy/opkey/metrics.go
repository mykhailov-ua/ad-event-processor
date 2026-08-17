// Package opkey implements regionproxy opkey helpers.
package opkey

var (
	depthFn func(float64)
	shedFn  func()
)

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
