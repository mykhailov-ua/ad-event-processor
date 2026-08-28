package reports

const (
	ClickhouseDimSub1Expr    = `nullIf(coalesce(nullIf(sub1, ''), nullIf(JSONExtractString(payload, 'sub1'), '')), '')`
	ClickhouseDimSub2Expr    = `nullIf(coalesce(nullIf(sub2, ''), nullIf(JSONExtractString(payload, 'sub2'), '')), '')`
	ClickhouseDimCountryExpr = `coalesce(nullIf(country, ''), nullIf(JSONExtractString(payload, 'country'), ''), 'ZZ')`
	clickhouseDimSub1Expr    = ClickhouseDimSub1Expr
	clickhouseDimSub2Expr    = ClickhouseDimSub2Expr
	clickhouseDimCountryExpr = ClickhouseDimCountryExpr
	clickhouseDimCityExpr    = `nullIf(coalesce(nullIf(city, ''), nullIf(JSONExtractString(payload, 'city'), '')), '')`
	clickhouseDimDeviceExpr  = `coalesce(nullIf(device_type, ''), nullIf(JSONExtractString(payload, 'device_type'), ''), nullIf(JSONExtractString(payload, 'device'), ''), 'unknown')`
	clickhouseDimKeywordExpr = `nullIf(coalesce(nullIf(keyword, ''), nullIf(JSONExtractString(payload, 'keyword'), '')), '')`
)
