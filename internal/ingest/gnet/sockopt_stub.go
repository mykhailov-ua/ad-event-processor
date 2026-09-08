//go:build !linux

package gnet

import (
	"ad-event-processor/internal/config"

	pkgnet "github.com/panjf2000/gnet/v2"
)

func applyInboundTCPHardening(_ pkgnet.Conn, _ *config.Config) {}
