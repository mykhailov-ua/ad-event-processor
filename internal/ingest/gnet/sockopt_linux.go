//go:build linux

package gnet

import (
	"ad-event-processor/internal/config"

	pkgnet "github.com/panjf2000/gnet/v2"
	"golang.org/x/sys/unix"
)

const tcpUserTimeout = 18

func applyInboundTCPHardening(c pkgnet.Conn, cfg *config.Config) {
	if c == nil || cfg == nil {
		return
	}
	addr := c.RemoteAddr()
	if addr == nil || addr.Network() != "tcp" {
		return
	}
	if cfg.HTTP1TCPLingerSec >= 0 {
		_ = c.SetLinger(cfg.HTTP1TCPLingerSec)
	}
	if cfg.HTTP1TCPUserTimeoutMs > 0 {
		_ = setTCPUserTimeout(c.Fd(), cfg.HTTP1TCPUserTimeoutMs)
	}
}

func setTCPUserTimeout(fd int, timeoutMs int) error {
	if fd < 0 || timeoutMs <= 0 {
		return nil
	}
	return unix.SetsockoptInt(fd, unix.IPPROTO_TCP, tcpUserTimeout, timeoutMs)
}
