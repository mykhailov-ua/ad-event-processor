package config_test

import (
	"testing"

	"espx/internal/config"

	"github.com/stretchr/testify/assert"
)

func TestLicenseRequiredFromEnv_defaultOffInDev(t *testing.T) {
	t.Setenv("ESPX_LICENSE_REQUIRED", "")
	t.Setenv("ESPX_PROFILE", "")
	assert.False(t, config.LicenseRequiredFromEnv())
}

func TestLicenseRequiredFromEnv_productionProfile(t *testing.T) {
	t.Setenv("ESPX_LICENSE_REQUIRED", "")
	t.Setenv("ESPX_PROFILE", "production")
	assert.True(t, config.LicenseRequiredFromEnv())
}

func TestLicenseRequiredFromEnv_explicit(t *testing.T) {
	t.Setenv("ESPX_LICENSE_REQUIRED", "1")
	assert.True(t, config.LicenseRequiredFromEnv())
}
