package controlplane

const (
	chDimSub1Expr    = `nullIf(coalesce(nullIf(sub1, ''), nullIf(JSONExtractString(payload, 'sub1'), '')), '')`
	chDimSub2Expr    = `nullIf(coalesce(nullIf(sub2, ''), nullIf(JSONExtractString(payload, 'sub2'), '')), '')`
	chDimCountryExpr = `coalesce(nullIf(country, ''), nullIf(JSONExtractString(payload, 'country'), ''), 'ZZ')`
	chDimCityExpr    = `nullIf(coalesce(nullIf(city, ''), nullIf(JSONExtractString(payload, 'city'), '')), '')`
	chDimDeviceExpr  = `coalesce(nullIf(device_type, ''), nullIf(JSONExtractString(payload, 'device_type'), ''), nullIf(JSONExtractString(payload, 'device'), ''), 'unknown')`
	chDimKeywordExpr = `nullIf(coalesce(nullIf(keyword, ''), nullIf(JSONExtractString(payload, 'keyword'), '')), '')`
)
