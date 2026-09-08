package domains

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFilterDomainHealthRows_healthyOmitsBurned(t *testing.T) {
	rows := []DomainHealthDTO{
		{Hostname: "ok.example", HealthStatus: "healthy"},
		{Hostname: "bad.example", HealthStatus: "healthy", PoolStatus: "banned"},
		{Hostname: "down.example", HealthStatus: "down"},
	}
	got := filterDomainHealthRows(rows, domainHealthFilterHealthy)
	require.Len(t, got, 1)
	require.Equal(t, "ok.example", got[0].Hostname)
}

func TestFilterDomainHealthRows_degradedIncludesDown(t *testing.T) {
	rows := []DomainHealthDTO{
		{Hostname: "deg.example", HealthStatus: "degraded"},
		{Hostname: "down.example", HealthStatus: "down"},
		{Hostname: "burned.example", HealthStatus: "down", PoolStatus: "banned"},
	}
	got := filterDomainHealthRows(rows, domainHealthFilterDegraded)
	require.Len(t, got, 2)
	require.Equal(t, "deg.example", got[0].Hostname)
	require.Equal(t, "down.example", got[1].Hostname)
}

func TestFilterDomainHealthRows_burned_holdout(t *testing.T) {
	rows := []DomainHealthDTO{
		{Hostname: "ok.example", HealthStatus: "healthy"},
		{Hostname: "burned.example", PoolStatus: "banned"},
	}
	got := filterDomainHealthRows(rows, domainHealthFilterBurned)
	require.Len(t, got, 1)
	require.Equal(t, "burned.example", got[0].Hostname)
}

func TestParseDomainHealthFilter_unknownDefaultsAll(t *testing.T) {
	require.Equal(t, domainHealthFilterAll, parseDomainHealthFilter(""))
	require.Equal(t, domainHealthFilterAll, parseDomainHealthFilter("invalid"))
}
