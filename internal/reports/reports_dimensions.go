package reports

const (
	clickhouseDimSub1Expr    = `nullIf(coalesce(nullIf(sub1, ''), nullIf(JSONExtractString(payload, 'sub1'), '')), '')`
	clickhouseDimSub2Expr    = `nullIf(coalesce(nullIf(sub2, ''), nullIf(JSONExtractString(payload, 'sub2'), '')), '')`
	clickhouseDimCountryExpr = `coalesce(nullIf(country, ''), nullIf(JSONExtractString(payload, 'country'), ''), 'ZZ')`
	clickhouseDimCityExpr    = `nullIf(coalesce(nullIf(city, ''), nullIf(JSONExtractString(payload, 'city'), '')), '')`
	clickhouseDimDeviceExpr  = `coalesce(nullIf(device_type, ''), nullIf(JSONExtractString(payload, 'device_type'), ''), nullIf(JSONExtractString(payload, 'device'), ''), 'unknown')`
	clickhouseDimKeywordExpr = `nullIf(coalesce(nullIf(keyword, ''), nullIf(JSONExtractString(payload, 'keyword'), '')), '')`
)
