package gnet

import (
	"bytes"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	pkgnet "github.com/panjf2000/gnet/v2"
)

var GnetHarnessRemoteAddr = gnetHarnessRemoteAddr

var gnetHarnessRemoteAddr = &net.TCPAddr{IP: net.IPv4(1, 1, 1, 1), Port: 1234}

type GnetHarnessConn struct {
	pkgnet.Conn
	mu             sync.Mutex
	closed         atomic.Bool
	inbound        []byte
	written        []byte
	responses      [][]byte
	ctx            any
	addr           net.Addr
	benchZeroAlloc bool
	onWake         func(*GnetHarnessConn)
}

func NewGnetHarnessConn(inbound []byte) *GnetHarnessConn {
	return &GnetHarnessConn{
		inbound: append([]byte(nil), inbound...),
		written: make([]byte, 0, 512),
		addr:    gnetHarnessRemoteAddr,
	}
}

func NewGnetBenchConn(inbound []byte) *GnetHarnessConn {
	c := NewGnetHarnessConn(inbound)
	c.benchZeroAlloc = true
	return c
}

func (c *GnetHarnessConn) Context() any     { return c.ctx }
func (c *GnetHarnessConn) SetContext(v any) { c.ctx = v }

func (c *GnetHarnessConn) Write(b []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.writeLocked(b)
}

func (c *GnetHarnessConn) writeLocked(b []byte) (int, error) {
	c.written = append(c.written[:0], b...)
	if c.benchZeroAlloc {
		return len(b), nil
	}
	cp := make([]byte, len(b))
	copy(cp, b)
	c.responses = append(c.responses, cp)
	return len(b), nil
}

func (c *GnetHarnessConn) AsyncWrite(buf []byte, callback pkgnet.AsyncCallback) error {
	c.mu.Lock()
	_, err := c.writeLocked(buf)
	c.mu.Unlock()
	if callback != nil {
		_ = callback(c, err)
	}
	return err
}

func (c *GnetHarnessConn) SetOnWake(fn func(*GnetHarnessConn)) {
	c.onWake = fn
}

func (c *GnetHarnessConn) Wake(callback pkgnet.AsyncCallback) error {
	if c.onWake != nil {
		c.onWake(c)
	}
	if callback != nil {
		_ = callback(c, nil)
	}
	return nil
}

func (c *GnetHarnessConn) InboundBuffered() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.inbound)
}

func (c *GnetHarnessConn) Peek(n int) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if n > len(c.inbound) {
		n = len(c.inbound)
	}
	return c.inbound[:n], nil
}

func (c *GnetHarnessConn) Discard(n int) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if n > len(c.inbound) {
		n = len(c.inbound)
	}
	c.inbound = c.inbound[n:]
	return n, nil
}

func (c *GnetHarnessConn) RemoteAddr() net.Addr {
	if c.addr != nil {
		return c.addr
	}
	return gnetHarnessRemoteAddr
}

func (c *GnetHarnessConn) SetInbound(b []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.inbound = append(c.inbound[:0], b...)
}

func (c *GnetHarnessConn) InboundBytes() []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]byte(nil), c.inbound...)
}

func (c *GnetHarnessConn) Append(b []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.inbound = append(c.inbound, b...)
}

func (c *GnetHarnessConn) Written() []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]byte(nil), c.written...)
}

func (c *GnetHarnessConn) ClearWritten() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.written = c.written[:0]
}

func (c *GnetHarnessConn) ClearResponses() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.responses = c.responses[:0]
}

func (c *GnetHarnessConn) WriteCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.responses)
}

func (c *GnetHarnessConn) AllResponses() [][]byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([][]byte, len(c.responses))
	for i, r := range c.responses {
		out[i] = append([]byte(nil), r...)
	}
	return out
}

func (c *GnetHarnessConn) SetRemoteAddr(addr net.Addr) { c.addr = addr }

func (c *GnetHarnessConn) SetReadDeadline(time.Time) error  { return nil }
func (c *GnetHarnessConn) SetWriteDeadline(time.Time) error { return nil }
func (c *GnetHarnessConn) SetDeadline(time.Time) error      { return nil }
func (c *GnetHarnessConn) Close() error {
	c.closed.Store(true)
	return nil
}

func (c *GnetHarnessConn) Closed() bool {
	return c.closed.Load()
}

func BuildGnetHTTP(method, path string, headers map[string]string, body []byte) []byte {
	var buf bytes.Buffer
	buf.WriteString(method)
	buf.WriteByte(' ')
	buf.WriteString(path)
	buf.WriteString(" HTTP/1.1\r\n")
	if _, ok := headers["Content-Length"]; !ok && body != nil {
		headers = copyGnetHarnessHeaders(headers)
		headers["Content-Length"] = strconv.Itoa(len(body))
	}
	for k, v := range headers {
		buf.WriteString(k)
		buf.WriteString(": ")
		buf.WriteString(v)
		buf.WriteString("\r\n")
	}
	buf.WriteString("\r\n")
	if len(body) > 0 {
		buf.Write(body)
	}
	return buf.Bytes()
}

func copyGnetHarnessHeaders(h map[string]string) map[string]string {
	out := make(map[string]string, len(h)+1)
	for k, v := range h {
		out[k] = v
	}
	return out
}

func BuildGnetPostTrackJSON(body []byte) []byte {
	return BuildGnetHTTP("POST", "/track", map[string]string{
		"Content-Type": "application/json",
		"Connection":   "keep-alive",
	}, body)
}

func BuildGnetGetHealth() []byte {
	return BuildGnetHTTP("GET", "/health", map[string]string{
		"Connection":     "keep-alive",
		"Content-Length": "0",
	}, nil)
}

func ServeGnetHarness(h *Server, inbound []byte) (pkgnet.Action, *GnetHarnessConn) {
	c := NewGnetHarnessConn(inbound)
	if h != nil {
		ctx := h.allocConnContext(c)
		ctx.WorkerID = 0
		c.SetContext(ctx)
	}
	return h.OnTraffic(c), c
}

func ParseGnetHTTPStatus(Resp []byte) int {
	if len(Resp) < 12 || !bytes.HasPrefix(Resp, []byte("HTTP/1.1 ")) {
		return 0
	}
	code := 0
	for i := 9; i < len(Resp) && Resp[i] != ' '; i++ {
		if Resp[i] >= '0' && Resp[i] <= '9' {
			code = code*10 + int(Resp[i]-'0')
		} else {
			break
		}
	}
	return code
}

func ParseGnetHTTPBody(Resp []byte) []byte {
	idx := bytes.Index(Resp, []byte("\r\n\r\n"))
	if idx < 0 {
		return nil
	}
	return Resp[idx+4:]
}

func PostOpenRTBBidGnet(h *Server, body []byte) (int, []byte) {
	headers := map[string]string{
		"Content-Type":   "application/json",
		"Content-Length": strconv.Itoa(len(body)),
		"Connection":     "keep-alive",
	}
	_, conn := ServeGnetHarness(h, BuildGnetHTTP("POST", "/openrtb/bid", headers, body))
	return ParseGnetHTTPStatus(conn.Written()), conn.Written()
}

func PostTrackGnet(h *Server, body []byte, contentType, accept string) (int, []byte) {
	headers := map[string]string{
		"Content-Type": contentType,
		"Connection":   "keep-alive",
	}
	if accept != "" {
		headers["Accept"] = accept
	}
	_, conn := ServeGnetHarness(h, BuildGnetHTTP("POST", "/track", headers, body))
	return ParseGnetHTTPStatus(conn.Written()), conn.Written()
}

func PostTrackGnetJSON(h *Server, body []byte) (int, []byte) {
	return PostTrackGnetJSONWait(h, body, 0)
}

func PostTrackGnetJSONWait(h *Server, body []byte, timeout time.Duration) (int, []byte) {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	_, conn := ServeGnetHarness(h, BuildGnetPostTrackJSON(body))
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		written := conn.Written()
		if len(written) > 0 {
			return ParseGnetHTTPStatus(written), written
		}
		time.Sleep(50 * time.Microsecond)
	}
	return 0, nil
}

func BuildGnetGetReady() []byte {
	return BuildGnetHTTP("GET", "/ready", map[string]string{
		"Connection":     "keep-alive",
		"Content-Length": "0",
	}, nil)
}

func GetReadyGnet(h *Server) (status int, body string) {
	_, conn := ServeGnetHarness(h, BuildGnetGetReady())
	return ParseGnetHTTPStatus(conn.Written()), string(ParseGnetHTTPBody(conn.Written()))
}

func GetHealthGnet(h *Server) (status int, body string) {
	_, conn := ServeGnetHarness(h, BuildGnetGetHealth())
	return ParseGnetHTTPStatus(conn.Written()), string(ParseGnetHTTPBody(conn.Written()))
}
