package export

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseLayerDesyncMinCount_defaultsAndClamps(t *testing.T) {
	assert.Equal(t, uint8(2), parseLayerDesyncMinCount(nil))
	assert.Equal(t, uint8(5), parseLayerDesyncMinCount(json.RawMessage(`{"layer_desync_count":5}`)))
	assert.Equal(t, uint8(255), parseLayerDesyncMinCount(json.RawMessage(`{"layer_desync_count":999}`)))
}
