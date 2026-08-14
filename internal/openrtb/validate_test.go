package openrtb

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validBidRequest26() []byte {
	return []byte(`{
  "id": "req-26-001",
  "imp": [{
    "id": "1",
    "bidfloor": 0.5,
    "banner": {"w": 300, "h": 250}
  }],
  "site": {"id": "site-1", "page": "https://example.com"},
  "device": {"ip": "203.0.113.1", "ua": "Mozilla/5.0"},
  "cur": ["USD"]
}`)
}

func TestValidateOpenRTB26_valid(t *testing.T) {
	res := ValidateBytes(validBidRequest26(), ExchangeConfig{MultiImpMax: 1})
	assert.True(t, res.Valid)
	assert.Equal(t, "2.6", res.Version)
	assert.Empty(t, res.Errors)
}

func TestValidateOpenRTB26_missingID(t *testing.T) {
	payload := []byte(`{"imp":[{"id":"1","bidfloor":0.5,"banner":{"w":300,"h":250}}],"site":{"page":"x"},"device":{"ip":"1.1.1.1","ua":"x"}}`)
	res := ValidateBytes(payload, ExchangeConfig{MultiImpMax: 1})
	assert.False(t, res.Valid)
	assert.Contains(t, res.Errors[0], "BidRequest.id is required")
}

func TestValidateOpenRTB26_missingImp(t *testing.T) {
	payload := []byte(`{"id":"x","imp":[],"site":{"page":"x"},"device":{"ip":"1.1.1.1","ua":"x"}}`)
	res := ValidateBytes(payload, ExchangeConfig{MultiImpMax: 1})
	assert.False(t, res.Valid)
	assert.Contains(t, res.Errors[0], "imp must contain at least one")
}

func TestValidateOpenRTB26_impMissingFormat(t *testing.T) {
	payload := []byte(`{"id":"x","imp":[{"id":"1","bidfloor":0.5}],"site":{"page":"x"},"device":{"ip":"1.1.1.1","ua":"x"}}`)
	res := ValidateBytes(payload, ExchangeConfig{MultiImpMax: 1})
	assert.False(t, res.Valid)
	assert.Contains(t, strings.Join(res.Errors, " "), "banner or video")
}

func TestValidateOpenRTB26_multipleInventory(t *testing.T) {
	payload := []byte(`{
	  "id":"x",
	  "imp":[{"id":"1","bidfloor":0.5,"banner":{"w":1,"h":1}}],
	  "site":{"id":"s"},
	  "app":{"id":"a"},
	  "device":{"ip":"1.1.1.1","ua":"x"}
	}`)
	res := ValidateBytes(payload, ExchangeConfig{MultiImpMax: 1})
	assert.False(t, res.Valid)
	assert.Contains(t, res.Errors[0], "at most one of site, app, or dooh")
}

func TestValidateOpenRTB26_invalidJSON(t *testing.T) {
	res := ValidateBytes([]byte(`{not json`), ExchangeConfig{})
	assert.False(t, res.Valid)
	assert.Equal(t, "invalid JSON", res.Errors[0])
}

func TestValidateOpenRTB26_errorCap(t *testing.T) {
	var b strings.Builder
	b.WriteString(`{"id":"","imp":[`)
	for i := range 60 {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(`{"id":""}`)
	}
	b.WriteString(`]}`)
	res := ValidateBytes([]byte(b.String()), ExchangeConfig{MultiImpMax: 100})
	assert.False(t, res.Valid)
	assert.LessOrEqual(t, len(res.Errors), maxValidationErrors)
}

func TestValidate_rejectsOpenRTB30Bytes(t *testing.T) {
	payload := []byte(`{"openrtb":{"request":{"id":"r1","item":[{"id":"1"}]}}}`)
	res := ValidateBytes(payload, ExchangeConfig{})
	assert.False(t, res.Valid)
	require.NotEmpty(t, res.Errors)
	assert.Contains(t, res.Errors[0], "3.0")
}

func TestValidateOpenRTB26_rejectsNonUSDEUR(t *testing.T) {
	payload := []byte(`{
	  "id": "r1",
	  "cur": ["GBP"],
	  "imp": [{"id": "1", "bidfloor": 1.0, "banner": {"w":1,"h":1}}],
	  "site": {"page": "https://example.com"},
	  "device": {"ip": "1.1.1.1", "ua": "x"}
	}`)
	res := ValidateBytes(payload, ExchangeConfig{MultiImpMax: 1})
	assert.False(t, res.Valid)
	assert.Contains(t, strings.Join(res.Errors, " "), "GBP")
}

func TestValidateOpenRTB26_allowsEUR(t *testing.T) {
	payload := []byte(`{
	  "id": "r1",
	  "cur": ["EUR"],
	  "imp": [{"id": "1", "bidfloor": 1.0, "banner": {"w":1,"h":1}}],
	  "site": {"page": "https://example.com"},
	  "device": {"ip": "1.1.1.1", "ua": "x"}
	}`)
	res := ValidateBytes(payload, ExchangeConfig{MultiImpMax: 1})
	assert.True(t, res.Valid)
}
