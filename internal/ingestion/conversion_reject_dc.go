package ingestion

type ConversionDatacenterIPChecker struct {
	geo GeoProvider
	dc  *DCASNTable
}

func NewConversionDatacenterIPChecker(geo GeoProvider, dc *DCASNTable) *ConversionDatacenterIPChecker {
	if geo == nil && dc == nil {
		return nil
	}
	return &ConversionDatacenterIPChecker{geo: geo, dc: dc}
}

func (c *ConversionDatacenterIPChecker) IsDatacenterIP(ip string) bool {
	if c == nil || ip == "" {
		return false
	}
	if c.geo != nil {
		anon, err := c.geo.IsAnonymous(ip)
		if err == nil && anon {
			return true
		}
	}
	if c.dc == nil || !c.dc.Ready() || c.geo == nil {
		return false
	}
	lookup, ok := c.geo.(ASNLookup)
	if !ok {
		return false
	}
	asn, asnOK := lookup.LookupASN(ip)
	if !asnOK || asn == 0 {
		return false
	}
	return c.dc.IsDatacenter(asn)
}
