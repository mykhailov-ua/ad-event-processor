package ingestion

import (
	"fmt"
	"strings"
)

const chaosValidTrackJSON = `{"type":"click","campaign_id":"550e8400-e29b-41d4-a716-446655440000"}`

func chaosWSBomb(n int, suffix string) []byte {
	var b strings.Builder
	b.Grow(n + len(suffix))
	for i := 0; i < n; i++ {
		b.WriteByte(' ')
	}
	b.WriteString(suffix)
	return []byte(b.String())
}

func chaosSlowBodyHeaders() []byte {
	return []byte("POST /track HTTP/1.1\r\nContent-Type: application/json\r\nContent-Length: 1048576\r\n\r\n")
}

func chaosSlowBodyPrefixBytes() []byte {
	return []byte(`{"type":"click","campaign_id":"550e8400-e29b-41d4-a716-446655440000","payload":{`)
}

func fragmentedChunkedOpenRTBRequest() []byte {
	body := []byte(`{"id":"req-1","imp":[{"id":"1"}]}`)
	half := len(body) / 2
	var wire []byte
	wire = append(wire, "POST /openrtb/bid HTTP/1.1\r\nTransfer-Encoding: chunked\r\nContent-Type: application/json\r\n\r\n"...)
	wire = append(wire, fmt.Sprintf("%x\r\n", half)...)
	wire = append(wire, body[:half]...)
	wire = append(wire, "\r\n"...)
	wire = append(wire, fmt.Sprintf("%x\r\n", len(body)-half)...)
	wire = append(wire, body[half:]...)
	wire = append(wire, "\r\n0\r\n\r\n"...)
	return wire
}
