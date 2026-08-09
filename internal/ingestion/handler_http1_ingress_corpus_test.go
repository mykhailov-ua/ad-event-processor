package ingestion

import (
	"fmt"
	"testing"
	"time"

	"espx/pkg/faultproof"

	"github.com/stretchr/testify/require"
)

func TestChaos_CrossHop_NginxGnet(t *testing.T) {
	corpus := buildNginxIngressCorpus()
	require.GreaterOrEqual(t, len(corpus), 200)

	var diffs []string
	start := time.Now()
	for _, tc := range corpus {
		edge, gnet, differential := http1IngressCanonical(tc.Wire, tc.MaxBody)
		if differential {
			diffs = append(diffs, fmt.Sprintf(
				"%s edge=%s/%d/%d gnet=%s/%d/%d",
				tc.Name, edge.Verdict, edge.BodyLen, edge.Consumed,
				gnet.Verdict, gnet.BodyLen, gnet.Consumed,
			))
		}
	}
	elapsed := time.Since(start)

	faultproof.Log(t, "parser_security_ps_g04", map[string]string{
		"gap_id":             "PS-G04",
		"gap":                boolGapStr(len(diffs) == 0),
		"corpus_cases":       fmt.Sprintf("%d", len(corpus)),
		"differential_count": fmt.Sprintf("%d", len(diffs)),
		"elapsed_ms":         fmt.Sprintf("%d", elapsed.Milliseconds()),
	})
	require.Empty(t, diffs, "nginx↔gnet ingress differentials")
	require.Less(t, elapsed, 5*time.Second)
}

func TestHTTP1IngressCanonical_trackRequiresCL(t *testing.T) {
	wire := []byte("POST /track HTTP/1.1\r\n\r\n")
	edge, gnet, diff := http1IngressCanonical(wire, 1024)
	require.False(t, diff)
	require.Equal(t, IngressReject, edge.Verdict)
	require.Equal(t, IngressReject, gnet.Verdict)
}

func TestHTTP1IngressCanonical_trackRejectsChunked(t *testing.T) {
	wire := []byte("POST /track HTTP/1.1\r\nTransfer-Encoding: chunked\r\n\r\n0\r\n\r\n")
	edge, gnet, diff := http1IngressCanonical(wire, 1024)
	require.False(t, diff)
	require.Equal(t, IngressReject, edge.Verdict)
	require.Equal(t, IngressReject, gnet.Verdict)
}

func boolGapStr(closed bool) string {
	if closed {
		return "closed"
	}
	return "open"
}
