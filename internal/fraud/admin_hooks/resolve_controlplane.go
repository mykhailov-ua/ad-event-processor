package admin_hooks

import (
	"context"
	"fmt"
	"time"
)

func NewManagementBlocker(baseURL, apiKey string, timeout time.Duration) (BlacklistBlocker, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("admin API key required for management client")
	}
	if baseURL == "" {
		return nil, fmt.Errorf("management base URL required")
	}
	return NewControlplaneClient(baseURL, apiKey, timeout), nil
}

func ResolveManagementBlocker(managementURL, apiKey string) (BlacklistBlocker, error) {
	return NewManagementBlocker(managementURL, apiKey, 10*time.Second)
}

func ResolveManagementBlockerFromConfig(managementURL, managementPort, apiKey string) (BlacklistBlocker, error) {
	if managementURL == "" {
		managementURL = "http://127.0.0.1:" + managementPort
	}
	return ResolveManagementBlocker(managementURL, apiKey)
}

func BlockIP(ctx context.Context, blocker BlacklistBlocker, ip string) error {
	if blocker == nil {
		return fmt.Errorf("blacklist blocker not configured")
	}
	return blocker.BlockIP(ctx, ip)
}
