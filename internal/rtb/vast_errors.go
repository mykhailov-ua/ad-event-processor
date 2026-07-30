package rtb

import "errors"

var (
	ErrVASTMalformed = errors.New("rtb: vast malformed")
	ErrVASTNoAds     = errors.New("rtb: vast has no ads")
)
