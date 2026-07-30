package management

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBilling_DomainMapped(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "billing", FileDomain("billing_money.go"))
}
