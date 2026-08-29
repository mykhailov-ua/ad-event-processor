package stream

import "ad-event-processor/internal/domain"

const ingressCostSourceMacro = "ingress_macro"

func clickAttributedCostSource(evt *domain.Event) string {
	if evt == nil || evt.IngressCostMicro <= 0 {
		return ""
	}
	return ingressCostSourceMacro
}
