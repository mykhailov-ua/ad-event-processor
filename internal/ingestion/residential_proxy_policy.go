package ingestion

type residentialProxyPolicy struct {
	ProxyMinEvents             float64
	ProxyMaxCTR                float64
	ProxyMinUsers              float64
	ProxyMinUserClickGap       float64
	ProxyMinEventsPerUser      float64
	ProxyMinImpressionPressure float64
	ProxyMinUsersPerUA         float64
	ProxyMinClicks             float64
}

type residentialProxyRow struct {
	Events      int
	Clicks      int
	UniqueUsers int
	UniqueUAs   int
}

func defaultResidentialProxyPolicy() residentialProxyPolicy {
	return residentialProxyPolicy{
		ProxyMinEvents:             80,
		ProxyMaxCTR:                0.05,
		ProxyMinUsers:              20,
		ProxyMinUserClickGap:       5.0,
		ProxyMinEventsPerUser:      5.0,
		ProxyMinImpressionPressure: 12.0,
		ProxyMinUsersPerUA:         2.5,
		ProxyMinClicks:             2,
	}
}

type residentialProxyMetrics struct {
	events             float64
	clicks             float64
	ctr                float64
	uniqueUsers        float64
	uniqueUAs          float64
	eventsPerUser      float64
	impressionPressure float64
	userClickGap       float64
	usersPerUA         float64
}

func residentialProxyMetricsFromRow(row residentialProxyRow) residentialProxyMetrics {
	events := float64(row.Events)
	clicks := float64(row.Clicks)
	users := float64(row.UniqueUsers)
	uas := float64(row.UniqueUAs)
	return residentialProxyMetrics{
		events:             events,
		clicks:             clicks,
		ctr:                safeRatio(clicks, events),
		uniqueUsers:        users,
		uniqueUAs:          uas,
		eventsPerUser:      safeRatio(events, users),
		impressionPressure: safeRatio(events, clicks+1),
		userClickGap:       safeRatio(users, clicks+1),
		usersPerUA:         safeRatio(users, uas),
	}
}

func safeRatio(numerator, denominator float64) float64 {
	if denominator <= 0 {
		return 0
	}
	return numerator / denominator
}

func residentialProxySignal(row residentialProxyRow, cfg residentialProxyPolicy) bool {
	m := residentialProxyMetricsFromRow(row)
	if m.events < cfg.ProxyMinEvents {
		return false
	}
	if m.ctr > cfg.ProxyMaxCTR {
		return false
	}
	if m.uniqueUsers < cfg.ProxyMinUsers {
		return false
	}
	if m.userClickGap < cfg.ProxyMinUserClickGap {
		return false
	}
	if m.eventsPerUser < cfg.ProxyMinEventsPerUser {
		return false
	}
	if m.impressionPressure < cfg.ProxyMinImpressionPressure {
		return false
	}
	if m.usersPerUA < cfg.ProxyMinUsersPerUA {
		return false
	}
	if m.clicks < cfg.ProxyMinClicks {
		return false
	}
	return true
}
