package proxyupstream

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"
)

var (
	ErrEmptyURL        = errors.New("proxy_upstream_url is required when click_delivery=proxy")
	ErrInvalidScheme   = errors.New("proxy_upstream_url must use https (or http only with lab flag)")
	ErrUserinfo        = errors.New("proxy_upstream_url must not contain userinfo")
	ErrFragment        = errors.New("proxy_upstream_url must not contain a fragment")
	ErrPrivateUpstream = errors.New("proxy_upstream_url must not target private, loopback, or link-local addresses")
	ErrLookupFailed    = errors.New("proxy_upstream_url host lookup failed")
)

const (
	ClickDeliveryRedirect = "redirect"
	ClickDeliveryProxy    = "proxy"
)

func isBlockedIP(ip net.IP) bool {
	if ip == nil || ip.IsUnspecified() || ip.IsLoopback() || ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() {
		return true
	}
	ip4 := ip.To4()
	if ip4 != nil && ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127 {
		return true
	}
	return false
}

func ValidateURL(ctx context.Context, raw string, allowHTTP bool) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ErrEmptyURL
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid proxy_upstream_url: %w", err)
	}
	if u.User != nil {
		return ErrUserinfo
	}
	if u.Fragment != "" {
		return ErrFragment
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
	case "http":
		if !allowHTTP {
			return ErrInvalidScheme
		}
	default:
		return ErrInvalidScheme
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("invalid proxy_upstream_url: missing host")
	}
	if ip := net.ParseIP(host); ip != nil {
		if isBlockedIP(ip) {
			return ErrPrivateUpstream
		}
		return nil
	}
	lookupCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupIPAddr(lookupCtx, host)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrLookupFailed, err)
	}
	if len(addrs) == 0 {
		return fmt.Errorf("%w: no addresses for %q", ErrLookupFailed, host)
	}
	for _, a := range addrs {
		if isBlockedIP(a.IP) {
			return ErrPrivateUpstream
		}
	}
	return nil
}

func ValidateDeliveryPair(ctx context.Context, delivery, upstream string, allowHTTP bool) error {
	delivery = strings.TrimSpace(delivery)
	if delivery == "" {
		delivery = ClickDeliveryRedirect
	}
	switch delivery {
	case ClickDeliveryRedirect:
		return nil
	case ClickDeliveryProxy:
		return ValidateURL(ctx, upstream, allowHTTP)
	default:
		return fmt.Errorf("invalid click_delivery %q", delivery)
	}
}
