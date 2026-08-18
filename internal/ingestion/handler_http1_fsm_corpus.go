package ingestion

var nginxTrackCorpus = []byte(
	"POST /track HTTP/1.1\r\n" +
		"Host: edge.local\r\n" +
		"Content-Type: application/json\r\n" +
		"Content-Length: 69\r\n" +
		"X-Forwarded-For: 203.0.113.10\r\n" +
		"X-Real-IP: 203.0.113.10\r\n" +
		"User-Agent: Mozilla/5.0\r\n" +
		"Accept: application/json\r\n" +
		"Accept-Language: en-US\r\n" +
		"X-TLS-Hash: abc123def456\r\n" +
		"Sec-CH-UA: \"Chromium\";v=\"120\"\r\n" +
		"Connection: keep-alive\r\n" +
		"\r\n" +
		`{"campaign_id":"00000000-0000-0000-0000-000000000001","type":"click"}`,
)
