package ads

import "time"

const (
	maxRetries  = 3
	initialWait = 100 * time.Millisecond
	maxWait     = 2 * time.Second
)
