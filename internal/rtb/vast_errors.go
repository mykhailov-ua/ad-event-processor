package rtb

import "errors"

var (
	// ErrVASTMalformed rejects empty or unparseable VAST payloads.
	ErrVASTMalformed = errors.New("rtb: vast malformed")
	// ErrVASTNoAds rejects documents without any <Ad> elements.
	ErrVASTNoAds = errors.New("rtb: vast has no ads")
)
