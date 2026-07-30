package config

import "time"

func LifecycleShutdownTimeout() time.Duration {
	return time.Duration(getEnvInt("SHUTDOWN_TIMEOUT_MS", 15000)) * time.Millisecond
}

func LifecycleWaitTimeout() time.Duration {
	return time.Duration(getEnvInt("WAIT_TIMEOUT_MS", 5000)) * time.Millisecond
}
