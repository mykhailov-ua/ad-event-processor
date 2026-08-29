package ingest

import (
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/internal/stream"
)

const ingressCostSourceMacro = "ingress_macro"

func attachIngressCost(evt *domain.Event, camp *domain.Campaign, parsed *clickQueryParsed) {
	if evt == nil || camp == nil || parsed == nil || !camp.IngressCost.Enabled() {
		return
	}
	raw := ingressCostQueryValue(camp.IngressCost.Param, parsed)
	if len(raw) == 0 {
		return
	}
	micro := parseIngressCostAmount(raw, camp.IngressCost.ScaleMicro)
	if micro <= 0 {
		return
	}
	if camp.IngressCost.MaxMicro > 0 && micro > camp.IngressCost.MaxMicro {
		return
	}
	evt.IngressCostMicro = micro
}

func ingressCostQueryValue(param domain.IngressCostParam, parsed *clickQueryParsed) []byte {
	switch param {
	case domain.IngressCostParamCost:
		return parsed.IngressCost
	case domain.IngressCostParamCPC:
		return parsed.IngressCPC
	case domain.IngressCostParamBid:
		return parsed.IngressBid
	default:
		return nil
	}
}

func parseIngressCostAmount(raw []byte, scaleMicro bool) int64 {
	if len(raw) == 0 {
		return 0
	}
	if scaleMicro {
		return parseIntegerMicro(raw)
	}
	return parseDecimalMicro(raw)
}

func parseIntegerMicro(raw []byte) int64 {
	i := 0
	n := len(raw)
	for i < n && (raw[i] == ' ' || raw[i] == '\t') {
		i++
	}
	if i >= n {
		return 0
	}
	var val int64
	for i < n {
		c := raw[i]
		if c < '0' || c > '9' {
			break
		}
		val = val*10 + int64(c-'0')
		i++
	}
	return val
}

func clickAttributedCostSource(evt *domain.Event) string {
	if evt == nil || evt.IngressCostMicro <= 0 {
		return ""
	}
	return ingressCostSourceMacro
}

func incIngressLegacyJSON() {
	metrics.IngressLegacyJSONTotal.Inc()
}

type (
	IngressQuotaCell = stream.IngressQuotaCell
	ingressQuotaMap  = stream.IngressQuotaMap
)

func buildIngressQuotaMap(epoch int64, limits *UDPControlLimits, numWorkers int) *ingressQuotaMap {
	return stream.BuildIngressQuotaMap(epoch, limits, numWorkers)
}

type unpaddedIngressCounters = stream.UnpaddedIngressCounters
