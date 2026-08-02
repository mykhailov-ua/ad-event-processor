package openrtb

import (
	"bytes"
	"encoding/json"
	"errors"
)

var (
	ErrEmptyBody    = errors.New("request body is empty")
	ErrInvalidJSON  = errors.New("invalid JSON")
	ErrOpenRTB30    = errors.New("OpenRTB 3.0 not supported; use 2.6")
	ErrBodyTooLarge = errors.New("request body exceeds limit")
)

func Decode(body []byte) (BidRequest, error) {
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return BidRequest{}, ErrEmptyBody
	}
	if !json.Valid(body) {
		return BidRequest{}, ErrInvalidJSON
	}
	if isOpenRTB30Shape(body) {
		return BidRequest{}, ErrOpenRTB30
	}
	var req BidRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return BidRequest{}, ErrInvalidJSON
	}
	return req, nil
}

func isOpenRTB30Shape(body []byte) bool {
	return bytes.Contains(body, []byte(`"openrtb"`)) &&
		bytes.Contains(body, []byte(`"request"`))
}

func IsOpenRTB30Shape(body []byte) bool {
	return isOpenRTB30Shape(body)
}

func ExchangeBodyPrecheck(body []byte) ValidationResult {
	body = trimBody(body)
	if len(body) == 0 {
		var c valCollector
		c.add("request body is empty")
		return c.result("")
	}
	if !jsonValid(body) {
		var c valCollector
		c.add("invalid JSON")
		return c.result("")
	}
	if isOpenRTB30Shape(body) {
		var c valCollector
		c.add("OpenRTB 3.0 is not supported; exchange accepts 2.6 only")
		return c.result("3.0")
	}
	return ValidationResult{Valid: true, Version: "2.6"}
}
