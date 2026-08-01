package notify

import (
	"regexp"
	"strings"
	"sync"

	"espx/pkg/branding"
)

var (
	adminBaseURLMu sync.RWMutex
	adminBaseURL   = branding.AdminConsoleURL()
	ipAddressRegex = regexp.MustCompile(`\b(?:[0-9]{1,3}\.){3}[0-9]{1,3}\b`)
)

func SetAdminBaseURL(baseURL string) {
	adminBaseURLMu.Lock()
	defer adminBaseURLMu.Unlock()
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		adminBaseURL = branding.AdminConsoleURL()
		return
	}
	adminBaseURL = baseURL
}

func currentAdminBaseURL() string {
	adminBaseURLMu.RLock()
	defer adminBaseURLMu.RUnlock()
	return adminBaseURL
}

type InteractiveActions struct {
	AcknowledgeURL string
	BlockIPURL     string
	BlockIP        string
}

func BuildInteractiveActions(notificationID, title, body string) InteractiveActions {
	base := currentAdminBaseURL()
	var actions InteractiveActions

	if notificationID != "" {
		actions.AcknowledgeURL = base + "/admin/acknowledge?id=" + notificationID
	}
	if ip := ipAddressRegex.FindString(body + " " + title); ip != "" {
		actions.BlockIP = ip
		actions.BlockIPURL = base + "/admin/blacklist?ip=" + ip + "&source=manual"
	}
	return actions
}
