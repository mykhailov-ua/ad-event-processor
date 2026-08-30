package broker

import "encoding/binary"

// brokerConcatLengthPrefixedMessages builds the mmap WAL produce blob: for each message,
// uvarint(byte_len) || payload. Matches dispatchBatch encoding in broker_producer.go so
// BrokerStreamConsumer and ParseBrokerPayloadStream share one wire layout.
//
// FraudBrokerSink reuses this helper for multi-record fraud batches (one Produce per partition).
//
// Verify: go test ./internal/stream/broker/ -short -run TestBrokerProducer_EnqueueAndFlush -count=1
func brokerConcatLengthPrefixedMessages(payloads [][]byte) []byte {
	if len(payloads) == 0 {
		return nil
	}
	var buf []byte
	var lenBuf [binary.MaxVarintLen64]byte
	for _, payload := range payloads {
		if len(payload) == 0 {
			continue
		}
		// uvarint length prefix; consumer reads via binary.Uvarint per frame.
		nLen := binary.PutUvarint(lenBuf[:], uint64(len(payload)))
		buf = append(buf, lenBuf[:nLen]...)
		buf = append(buf, payload...)
	}
	// Empty slice input yields nil (caller treats as no-op Produce).
	return buf
}
