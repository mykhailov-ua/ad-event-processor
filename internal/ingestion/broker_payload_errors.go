package ingestion

import "errors"

var ErrBrokerPayloadUnrecognized = errors.New("unrecognized broker payload format")

var ErrFraudBrokerSinkConfig = errors.New("fraud broker sink requires broker addr and topic")
