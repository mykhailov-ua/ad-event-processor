package postback

import (
	"encoding/json"

	"github.com/bidshard/ad-event-processor/pkg/money"

	"github.com/google/uuid"
)

type PostbackPayload struct {
	CustomerID    uuid.UUID `json:"customer_id"`
	CampaignID    uuid.UUID `json:"campaign_id"`
	ClickID       string    `json:"click_id"`
	EventType     string    `json:"event_type"`
	PayoutMicro   int64     `json:"payout_micro"`
	TxID          string    `json:"tx_id"`
	SubID1        string    `json:"subid1"`
	Param10       string    `json:"param10"`
	Email         string    `json:"email"`
	Phone         string    `json:"phone"`
	FBCLID        string    `json:"fbclid"`
	GCLID         string    `json:"gclid"`
	TTCLID        string    `json:"ttclid"`
	TestEventCode string    `json:"test_event_code,omitempty"`
	subSlots      [maxSubMacroSlots]string
}

func (p *PostbackPayload) SubIDs() [maxSubMacroSlots]string {
	var out [maxSubMacroSlots]string
	copy(out[:], p.subSlots[:])
	if out[0] == "" {
		out[0] = p.SubID1
	}
	if out[9] == "" {
		out[9] = p.Param10
	}
	return out
}

func (p *PostbackPayload) UnmarshalJSON(data []byte) error {
	type payloadAlias PostbackPayload
	aux := struct {
		payloadAlias
		PayoutLegacy *float64 `json:"payout"`
	}{}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*p = PostbackPayload(aux.payloadAlias)
	if p.PayoutMicro == 0 && aux.PayoutLegacy != nil {
		micro, err := money.LegacyFloatToMicro(*aux.PayoutLegacy)
		if err != nil {
			return err
		}
		p.PayoutMicro = micro
	}
	var raw map[string]string
	if err := json.Unmarshal(data, &raw); err == nil {
		for i := 1; i <= maxSubMacroSlots; i++ {
			if v, ok := raw[subIDJSONKey(i, true)]; ok {
				p.subSlots[i-1] = v
			} else if v, ok := raw[subIDJSONKey(i, false)]; ok {
				p.subSlots[i-1] = v
			}
		}
		if p.SubID1 != "" {
			p.subSlots[0] = p.SubID1
		}
		if p.Param10 != "" {
			p.subSlots[9] = p.Param10
		}
	}
	return nil
}

func subIDJSONKey(idx int, subidStyle bool) string {
	prefix := "sub"
	if subidStyle {
		prefix = "subid"
	}
	if idx < 10 {
		return prefix + string(byte('0'+idx))
	}
	return prefix + string(byte('0'+idx/10)) + string(byte('0'+idx%10))
}

func (p *PostbackPayload) PayoutDollarsAPI() float64 {
	return money.APIValueFloat(p.PayoutMicro)
}
