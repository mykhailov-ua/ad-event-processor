package doctor

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"ad-event-processor/pkg/platformconfig"
)

type DNSProbe struct {
	Hostname string
}

func (p DNSProbe) Name() string { return "dns" }

func (p DNSProbe) Run(ctx context.Context) Result {
	start := time.Now()
	host := platformconfig.ResolveHost(p.Hostname)
	if host == "" {
		return Result{Name: "dns", Status: StatusSkip, Detail: "tracking_domain not set", Latency: time.Since(start).Milliseconds()}
	}

	var resolver net.Resolver
	hasA, errA := lookupHasRecords(ctx, &resolver, "ip4", host)
	hasAAAA, errAAAA := lookupHasRecords(ctx, &resolver, "ip6", host)

	if !hasA && !hasAAAA {
		var parts []string
		if errA != nil {
			parts = append(parts, fmt.Sprintf("A: %v", errA))
		}
		if errAAAA != nil {
			parts = append(parts, fmt.Sprintf("AAAA: %v", errAAAA))
		}
		detail := "no A/AAAA records"
		if len(parts) > 0 {
			detail = strings.Join(parts, "; ")
		}
		return Result{Name: "dns", Status: StatusFail, Detail: detail, Latency: time.Since(start).Milliseconds()}
	}

	status := StatusPass
	detail := fmt.Sprintf("A=%t AAAA=%t for %s", hasA, hasAAAA, host)
	if !hasA || !hasAAAA {
		status = StatusWarn
		detail = fmt.Sprintf("partial: A=%t AAAA=%t for %s", hasA, hasAAAA, host)
	}
	return Result{Name: "dns", Status: status, Detail: detail, Latency: time.Since(start).Milliseconds()}
}

func lookupHasRecords(ctx context.Context, resolver *net.Resolver, network, host string) (bool, error) {
	ips, err := resolver.LookupIP(ctx, network, host)
	if err != nil {
		return false, err
	}
	return len(ips) > 0, nil
}
