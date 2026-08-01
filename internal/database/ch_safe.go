package database

import (
	"regexp"
	"unicode"
)

var gaqlDateRE = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
var hexTokenRE = regexp.MustCompile(`^[0-9a-f]+$`)

// ValidClickHouseIdentifier rejects identifiers that could break out of SQL quoting.
func ValidClickHouseIdentifier(name string) bool {
	if name == "" || len(name) > 128 {
		return false
	}
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			continue
		}
		return false
	}
	return true
}

func ValidGAQLDate(date string) bool {
	return gaqlDateRE.MatchString(date)
}

func ValidCHHexToken(token string) bool {
	if token == "" || len(token) > 128 {
		return false
	}
	return hexTokenRE.MatchString(token)
}

func ClampCHLookbackHours(hours int) int {
	if hours < 1 {
		return 1
	}
	if hours > 24*90 {
		return 24 * 90
	}
	return hours
}

func ClampCHWindowSeconds(sec int64) int64 {
	if sec <= 0 {
		return 3600
	}
	if sec > 24*3600 {
		return 24 * 3600
	}
	return sec
}

func ClampCHBucketMicro(micro int64) int64 {
	if micro <= 0 {
		return 10_000
	}
	if micro > 1_000_000_000 {
		return 1_000_000_000
	}
	return micro
}
