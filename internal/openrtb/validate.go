package openrtb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

const maxValidationErrors = 50

var allowedCurrencies = map[string]struct{}{
	"USD": {},
	"EUR": {},
}

func Validate(req BidRequest, cfg ExchangeConfig) ValidationResult {
	var c valCollector
	if strings.TrimSpace(req.ID) == "" {
		c.add("BidRequest.id is required")
	}
	if len(req.Imp) == 0 {
		c.add("BidRequest.imp must contain at least one Imp object")
	}
	if cfg.MultiImpMax > 0 && len(req.Imp) > cfg.MultiImpMax {
		c.add("BidRequest.imp exceeds RTB_EXCHANGE_MULTI_IMP_MAX (%d)", cfg.MultiImpMax)
	}

	invCount := 0
	if req.Site != nil {
		invCount++
	}
	if req.App != nil {
		invCount++
	}
	if req.DOOH != nil {
		invCount++
		c.add("dooh inventory is not supported")
	}
	if invCount == 0 {
		c.add("BidRequest must contain site or app")
	}
	if invCount > 1 {
		c.add("BidRequest must contain at most one of site, app, or dooh")
	}

	if strings.TrimSpace(req.Device.IP) == "" && strings.TrimSpace(req.Device.IPv6) == "" {
		c.add("device.ip or device.ipv6 is required")
	}
	if strings.TrimSpace(req.Device.UA) == "" {
		c.add("device.ua is required")
	}

	validateCurrencyList(&c, req.Cur, "BidRequest.cur")

	for i, imp := range req.Imp {
		prefix := fmt.Sprintf("imp[%d]", i)
		if strings.TrimSpace(imp.ID) == "" {
			c.add("%s.id is required", prefix)
		}
		if imp.Audio != nil {
			c.add("%s.audio is not supported", prefix)
		}
		if imp.Native != nil {
			c.add("%s.native is not supported", prefix)
		}
		if imp.Banner == nil && imp.Video == nil {
			c.add("%s must include banner or video", prefix)
		}
		if imp.BidFloor < 0 {
			c.add("%s.bidfloor must be non-negative", prefix)
		}
		if cur := strings.TrimSpace(imp.BidFloorCur); cur != "" {
			validateCurrency(&c, cur, prefix+".bidfloorcur")
		}
	}

	return c.result("2.6")
}

func ValidateBytes(body []byte, cfg ExchangeConfig) ValidationResult {
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
	req, err := Decode(body)
	if err != nil {
		var c valCollector
		c.add("invalid OpenRTB 2.6 JSON")
		return c.result("2.6")
	}
	return Validate(req, cfg)
}

type valCollector struct {
	errors []string
}

func (c *valCollector) add(format string, args ...any) {
	if len(c.errors) >= maxValidationErrors {
		return
	}
	c.errors = append(c.errors, fmt.Sprintf(format, args...))
}

func (c *valCollector) result(version string) ValidationResult {
	return ValidationResult{
		Valid:   len(c.errors) == 0,
		Version: version,
		Errors:  c.errors,
	}
}

func validateCurrencyList(c *valCollector, currencies []string, field string) {
	for i, cur := range currencies {
		validateCurrency(c, cur, fmt.Sprintf("%s[%d]", field, i))
	}
}

func validateCurrency(c *valCollector, cur, field string) {
	cur = strings.ToUpper(strings.TrimSpace(cur))
	if cur == "" {
		return
	}
	if _, ok := allowedCurrencies[cur]; !ok {
		c.add("%s currency %q is not allowed; only USD and EUR are accepted", field, cur)
	}
}

func trimBody(body []byte) []byte {
	return bytes.TrimSpace(body)
}

func jsonValid(body []byte) bool {
	return json.Valid(body)
}
