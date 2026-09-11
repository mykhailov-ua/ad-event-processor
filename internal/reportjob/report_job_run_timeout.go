package reportjob

import (
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	reportJobRunTimeoutDefaultSec         = 120 // csv/json/zip/download
	reportJobRunTimeoutExtendedDefaultSec = 600 // xlsx/google_sheet
)

func reportJobRunTimeoutExtendedSec() int {
	if v := strings.TrimSpace(os.Getenv("REPORT_JOB_RUN_TIMEOUT_SEC")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return reportJobRunTimeoutExtendedDefaultSec
}

func reportJobNeedsExtendedRunTimeout(spec ReportJobSpec) bool {
	destination := strings.TrimSpace(spec.Destination)
	if destination == "" {
		destination = "download"
	}
	if destination == "google_sheet" {
		return true
	}
	format := strings.TrimSpace(spec.Format)
	if format == "" {
		format = "csv"
	}
	return format == "xlsx"
}

func reportJobRunTimeoutForSpec(spec ReportJobSpec) time.Duration {
	if reportJobNeedsExtendedRunTimeout(spec) {
		return time.Duration(reportJobRunTimeoutExtendedSec()) * time.Second
	}
	return time.Duration(reportJobRunTimeoutDefaultSec) * time.Second
}
