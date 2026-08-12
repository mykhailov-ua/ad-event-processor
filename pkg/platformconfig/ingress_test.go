package platformconfig_test

import (
	"testing"

	"github.com/bidshard/ad-event-processor/pkg/naming"
	"github.com/bidshard/ad-event-processor/pkg/platformconfig"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeIngressSchema(t *testing.T) {
	assert.Equal(t, platformconfig.IngressAdEventProcessorNative,
		platformconfig.NormalizeIngressSchema(naming.DeprecatedIngressNativeSchema()))
	assert.Equal(t, platformconfig.IngressAdEventProcessorNative,
		platformconfig.NormalizeIngressSchema("native_v1"))
	assert.Equal(t, platformconfig.IngressOpenRTB3,
		platformconfig.NormalizeIngressSchema(platformconfig.IngressOpenRTB3))
}

func TestDefaultIngressSchema(t *testing.T) {
	cfg := platformconfig.Default()
	assert.Equal(t, platformconfig.IngressAdEventProcessorNative, cfg.IngressSchema)
}
