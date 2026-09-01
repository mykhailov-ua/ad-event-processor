package platformadmin

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCustomerListOrderClause_mapsSortableFields(t *testing.T) {
	t.Parallel()

	assert.True(t, strings.Contains(customerListOrderClause("name", "asc"), "c.name ASC"))
	assert.True(t, strings.Contains(customerListOrderClause("balance", "desc"), "c.balance DESC"))
	assert.True(t, strings.Contains(customerListOrderClause("active_campaigns", "asc"), "active_campaigns"))
	assert.True(t, strings.Contains(customerListOrderClause("created_at", "desc"), "c.created_at DESC"))
}

func TestCustomerListOrderClause_holdoutRejectsInvalidOrder(t *testing.T) {
	t.Parallel()

	clause := customerListOrderClause("balance", "sideways")
	assert.True(t, strings.HasSuffix(clause, "DESC"))
}
