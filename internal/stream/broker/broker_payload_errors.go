package broker

import "errors"

// Broker payload decode errors (consumer and fraud sink wire-up). Distinct from producer
// ErrRingBufferFull which is hot-path admission/post-debit enqueue.

// ErrBrokerPayloadUnrecognized: Fetch payload is neither AdStreamEvent nor AdLogRecord VT.
// Live consumer skips with metric; shadow mode must not commit offset past corrupt frames
// (see broker_consumer.go and TestFault_BrokerLiveConsumer_CorruptPayload).
var ErrBrokerPayloadUnrecognized = errors.New("unrecognized broker payload format")

// ErrFraudBrokerSinkConfig: FraudBrokerSink.Produce requires broker addr + topic at wire-up.
var ErrFraudBrokerSinkConfig = errors.New("fraud broker sink requires broker addr and topic")
