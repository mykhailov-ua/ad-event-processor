package ingest

const chaosValidTrackJSON = `{"campaign_id":"00000000-0000-0000-0000-000000000001","type":"click"}`

func chaosSlowBodyHeaders() []byte {
	return []byte("POST /track HTTP/1.1\r\nHost: localhost\r\nContent-Type: application/json\r\nContent-Length: 128\r\n\r\n")
}

func chaosSlowBodyPrefixBytes() []byte {
	return []byte(`{"campaign_id":"00000000-0000-0000-0000-000000000001","type":"click","user_id":"`)
}
