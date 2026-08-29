package gnet

import pkgnet "github.com/panjf2000/gnet/v2"

type Reactor interface {
	React(req Request, c pkgnet.Conn) pkgnet.Action
}
