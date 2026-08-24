package controlplane

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

var errInvalidCronExpr = fmt.Errorf("invalid cron expression")

func validateReportCronExpr(expr string) error {
	fields := strings.Fields(strings.TrimSpace(expr))
	if len(fields) != 5 {
		return errInvalidCronExpr
	}
	for i, field := range fields {
		if field == "" {
			return errInvalidCronExpr
		}
		if err := validateCronField(field, i); err != nil {
			return err
		}
	}
	return nil
}

func validateCronField(field string, fieldIndex int) error {
	parts := strings.Split(field, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return errInvalidCronExpr
		}
		if strings.Contains(part, "/") {
			chunk, step, ok := strings.Cut(part, "/")
			if !ok || step == "" {
				return errInvalidCronExpr
			}
			if _, err := strconv.Atoi(step); err != nil {
				return errInvalidCronExpr
			}
			part = chunk
		}
		if part == "*" {
			continue
		}
		if strings.Contains(part, "-") {
			low, high, ok := strings.Cut(part, "-")
			if !ok {
				return errInvalidCronExpr
			}
			if err := validateCronValue(low, fieldIndex); err != nil {
				return err
			}
			if err := validateCronValue(high, fieldIndex); err != nil {
				return err
			}
			continue
		}
		if err := validateCronValue(part, fieldIndex); err != nil {
			return err
		}
	}
	return nil
}

func validateCronValue(raw string, fieldIndex int) error {
	if raw == "*" {
		return nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return errInvalidCronExpr
	}
	switch fieldIndex {
	case 0:
		if n < 0 || n > 59 {
			return errInvalidCronExpr
		}
	case 1:
		if n < 0 || n > 23 {
			return errInvalidCronExpr
		}
	case 2:
		if n < 1 || n > 31 {
			return errInvalidCronExpr
		}
	case 3:
		if n < 1 || n > 12 {
			return errInvalidCronExpr
		}
	case 4:
		if n < 0 || n > 7 {
			return errInvalidCronExpr
		}
	}
	return nil
}

func nextReportCronRun(expr string, after time.Time) (time.Time, error) {
	if err := validateReportCronExpr(expr); err != nil {
		return time.Time{}, err
	}
	fields := strings.Fields(strings.TrimSpace(expr))
	start := after.UTC().Truncate(time.Minute).Add(time.Minute)
	deadline := start.Add(366 * 24 * time.Hour)
	for t := start; t.Before(deadline); t = t.Add(time.Minute) {
		if cronMatches(fields, t) {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cron next run not found within horizon")
}

func cronMatches(fields []string, t time.Time) bool {
	dow := int(t.Weekday())
	return cronFieldMatches(fields[0], t.Minute()) &&
		cronFieldMatches(fields[1], t.Hour()) &&
		cronFieldMatches(fields[2], t.Day()) &&
		cronFieldMatches(fields[3], int(t.Month())) &&
		cronFieldMatches(fields[4], dow)
}

func cronFieldMatches(field string, value int) bool {
	for _, part := range strings.Split(field, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		step := 1
		if chunk, s, ok := strings.Cut(part, "/"); ok {
			if parsed, err := strconv.Atoi(s); err == nil && parsed > 0 {
				step = parsed
			}
			part = chunk
		}
		if part == "*" {
			return true
		}
		if low, high, ok := strings.Cut(part, "-"); ok {
			lo, err1 := strconv.Atoi(low)
			hi, err2 := strconv.Atoi(high)
			if err1 != nil || err2 != nil {
				continue
			}
			for v := lo; v <= hi; v += step {
				if v == value {
					return true
				}
			}
			continue
		}
		if n, err := strconv.Atoi(part); err == nil && n == value {
			return true
		}
	}
	return false
}
