package ingest

import (
	"net"
	"net/http"
	"testing"
)

func FuzzExtractClientIP_XFF(f *testing.F) {
	f.Add("203.0.113.1:1234", "198.51.100.99, 203.0.113.2", "10.0.0.0/8", byte(1))
	f.Add("10.0.0.5:443", "192.168.0.1", "10.0.0.0/8", byte(1))
	f.Add("8.8.8.8:53", "", "", byte(0))

	f.Fuzz(func(t *testing.T, remoteAddr, xff, trustedCIDR string, useTrusted byte) {
		req, err := http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.Skip()
		}
		if remoteAddr != "" {
			req.RemoteAddr = remoteAddr
		}
		if xff != "" {
			req.Header.Set("X-Forwarded-For", xff)
		}
		var trusted []string
		if useTrusted&1 != 0 && trustedCIDR != "" {
			trusted = []string{trustedCIDR}
		}
		ip := extractClientIP(req, trusted)
		if ip != "" {
			_ = net.ParseIP(ip)
		}
	})
}

func FuzzExtractClientIPGnet_XFF(f *testing.F) {
	f.Add("10.0.0.1:1234", "203.0.113.10", "10.0.0.0/8", byte(1))
	f.Add("1.1.1.1:443", "bad,,,xff", "1.1.1.1", byte(1))

	f.Fuzz(func(t *testing.T, remoteHostPort, xff, trusted string, useTrusted byte) {
		req := parsedHTTPRequest{}
		if xff != "" {
			req.ClientIP = []byte(xff)
		}
		host, port, err := net.SplitHostPort(remoteHostPort)
		if err != nil {
			host = remoteHostPort
			port = "1234"
		}
		ip := net.ParseIP(host)
		if ip == nil {
			t.Skip()
		}
		p, _ := net.LookupPort("tcp", port)
		addr := &net.TCPAddr{IP: ip, Port: p}
		ctx := &connContext{}
		conn := NewGnetHarnessConn(nil)
		conn.SetContext(ctx)
		conn.SetRemoteAddr(addr)

		var trustedProxies []string
		if useTrusted&1 != 0 && trusted != "" {
			trustedProxies = []string{trusted}
		}
		_ = extractClientIPGnet(ctx, &req, conn, trustedProxies)
	})
}
