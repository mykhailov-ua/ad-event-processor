package gnet

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Holdout: revert aliases AsyncWrite buffer; production clones via cloneAsyncWriteBytes before arena release.
func TestGnetHarnessConn_AsyncWrite_holdoutCopiesBuffer(t *testing.T) {
	conn := NewGnetHarnessConn(nil)
	buf := []byte("HTTP/1.1 200 OK\r\n\r\n")
	require.NoError(t, conn.AsyncWrite(buf, nil))
	buf[0] = 'X'

	responses := conn.AllResponses()
	require.Len(t, responses, 1)
	require.Equal(t, byte('H'), responses[0][0])
	require.Equal(t, "HTTP/1.1 200 OK\r\n\r\n", string(responses[0]))
}

func TestGnetBenchConn_AsyncWrite_skipsResponseSnapshot(t *testing.T) {
	conn := NewGnetBenchConn(nil)
	buf := []byte("bench")
	require.NoError(t, conn.AsyncWrite(buf, nil))
	require.Empty(t, conn.AllResponses())
	require.Equal(t, []byte("bench"), conn.Written())
}
