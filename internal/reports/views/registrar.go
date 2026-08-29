package views

import "errors"

var ErrForbidden = errors.New("forbidden")

type validationError string

func (e validationError) Error() string { return string(e) }

func validationErr(msg string) error { return validationError(msg) }

var liveReportExportKeys func() []string

func SetLiveReportExportKeys(fn func() []string) {
	liveReportExportKeys = fn
}
