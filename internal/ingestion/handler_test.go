package ingestion

import (
	"net"
	"sync"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/google/uuid"
	"github.com/panjf2000/gnet/v2"
)

var staticRemoteAddr = &net.TCPAddr{IP: net.IPv4(1, 1, 1, 1), Port: 1234}

type mockGnetConn struct {
	gnet.Conn
	mu      sync.Mutex
	written []byte
	ctx     any
}

func (m *mockGnetConn) Context() any     { return m.ctx }
func (m *mockGnetConn) SetContext(v any) { m.ctx = v }

func (m *mockGnetConn) Write(b []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.written = append(m.written[:0], b...)
	return len(b), nil
}

func (m *mockGnetConn) AsyncWrite(buf []byte, callback gnet.AsyncCallback) error {
	m.mu.Lock()
	_, err := m.writeLocked(buf)
	m.mu.Unlock()
	if callback != nil {
		_ = callback(m, err)
	}
	return err
}

func (m *mockGnetConn) writeLocked(b []byte) (int, error) {
	m.written = append(m.written[:0], b...)
	return len(b), nil
}

func (m *mockGnetConn) snapshotWritten() []byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]byte(nil), m.written...)
}

func waitMockGnetWritten(conn *mockGnetConn, timeout time.Duration) []byte {
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if written := conn.snapshotWritten(); len(written) > 0 {
			return written
		}
		time.Sleep(time.Millisecond)
	}
	return nil
}

func (m *mockGnetConn) RemoteAddr() net.Addr {
	return staticRemoteAddr
}

func BenchmarkAdsPacketHandlerJSON(b *testing.B) {
	cfg := &config.Config{
		MaxRequestBodySize: 1024 * 1024,
	}
	registry := &mockRegistry{}
	sharder := NewJumpHashSharder(1)
	handler := NewAdsPacketHandler(cfg, registry, nil, nil, nil, sharder, "fraud-stream", nil)

	payload := []byte(`{"campaign_id":"` + uuid.NewString() + `","user_id":"user123","type":"click","click_id":"click123","payload":{}}`)
	req := parsedHTTPRequest{
		Method:           []byte("POST"),
		Path:             []byte("/track"),
		ContentType:      []byte("application/json"),
		ClientIP:         []byte("1.1.1.1"),
		UserAgent:        []byte("Mozilla/5.0"),
		Body:             payload,
		ContentLength:    len(payload),
		HasContentLength: true,
	}

	conn := &mockGnetConn{written: make([]byte, 0, 512)}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.React(req, conn)
	}
}
