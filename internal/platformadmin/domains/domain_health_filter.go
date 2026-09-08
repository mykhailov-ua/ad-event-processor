package domains

type domainHealthFilter string

const (
	domainHealthFilterAll      domainHealthFilter = "all"
	domainHealthFilterHealthy  domainHealthFilter = "healthy"
	domainHealthFilterDegraded domainHealthFilter = "degraded"
	domainHealthFilterBurned   domainHealthFilter = "burned"
)

func parseDomainHealthFilter(raw string) domainHealthFilter {
	switch raw {
	case string(domainHealthFilterHealthy):
		return domainHealthFilterHealthy
	case string(domainHealthFilterDegraded):
		return domainHealthFilterDegraded
	case string(domainHealthFilterBurned):
		return domainHealthFilterBurned
	default:
		return domainHealthFilterAll
	}
}

func matchesDomainHealthFilter(dto DomainHealthDTO, filter domainHealthFilter) bool {
	switch filter {
	case domainHealthFilterAll:
		return true
	case domainHealthFilterBurned:
		return dto.PoolStatus == "banned"
	case domainHealthFilterHealthy:
		return dto.HealthStatus == "healthy" && dto.PoolStatus != "banned"
	case domainHealthFilterDegraded:
		return (dto.HealthStatus == "degraded" || dto.HealthStatus == "down") && dto.PoolStatus != "banned"
	default:
		return true
	}
}

func filterDomainHealthRows(rows []DomainHealthDTO, filter domainHealthFilter) []DomainHealthDTO {
	if filter == domainHealthFilterAll {
		return rows
	}
	out := make([]DomainHealthDTO, 0, len(rows))
	for _, row := range rows {
		if matchesDomainHealthFilter(row, filter) {
			out = append(out, row)
		}
	}
	return out
}
