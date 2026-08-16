package ingestion

import (
	"strings"
	"time"
)

// countryPrimaryTimezone maps ISO-3166 alpha-2 codes to a representative IANA zone.
// Cold-path attestation only (POST /track/verify).
var countryPrimaryTimezone = map[string]string{
	"US": "America/New_York",
	"GB": "Europe/London",
	"DE": "Europe/Berlin",
	"FR": "Europe/Paris",
	"UA": "Europe/Kyiv",
	"RU": "Europe/Moscow",
	"IN": "Asia/Kolkata",
	"JP": "Asia/Tokyo",
	"AU": "Australia/Sydney",
	"BR": "America/Sao_Paulo",
	"CA": "America/Toronto",
	"NL": "Europe/Amsterdam",
	"PL": "Europe/Warsaw",
	"IT": "Europe/Rome",
	"ES": "Europe/Madrid",
}

func timezoneOffsetHours(tz string, ts time.Time) (int, bool) {
	if tz == "" {
		return 0, false
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return 0, false
	}
	_, offset := ts.In(loc).Zone()
	return offset / 3600, true
}

func timezoneMismatchHours(browserTZ, country string, now time.Time) (mismatch bool, deltaHours int) {
	expected, ok := countryPrimaryTimezone[strings.ToUpper(country)]
	if !ok || browserTZ == "" {
		return false, 0
	}
	expOff, okExp := timezoneOffsetHours(expected, now)
	gotOff, okGot := timezoneOffsetHours(browserTZ, now)
	if !okExp || !okGot {
		return false, 0
	}
	delta := expOff - gotOff
	if delta < 0 {
		delta = -delta
	}
	return delta > 2, delta
}
