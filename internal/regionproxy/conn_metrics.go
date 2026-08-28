package regionproxy

var recordConnIdleClose func(reason string)

func BindConnIdleMetrics(record func(reason string)) {
	recordConnIdleClose = record
}
