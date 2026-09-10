package signing

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSignGETURL_stable(t *testing.T) {
	t.Parallel()
	secret := []byte("test-secret")
	sig := SignGETURL(secret, "https://example.com/pb?click_id=abc&sig=old")
	assert.NotEmpty(t, sig)
	assert.Equal(t, sig, SignGETURL(secret, "https://example.com/pb?click_id=abc&sig=other"))
}
