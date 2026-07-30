package database

import (
	"context"
	"errors"
	"strings"
)

func IsPoolClosedError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "closed pool")
}

func IsShutdownError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return true
	}
	if IsPoolClosedError(err) {
		return true
	}
	return strings.Contains(err.Error(), "client is closed")
}
