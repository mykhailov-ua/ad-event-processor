package budget

import (
	"encoding/json"
	"strings"
)

type IngressCostParam uint8

const (
	IngressCostDisabled IngressCostParam = iota
	IngressCostParamCost
	IngressCostParamCPC
	IngressCostParamBid
)

type IngressCostPolicy uint8

const (
	IngressCostPolicyIgnore IngressCostPolicy = iota
	IngressCostPolicyReject
)

type IngressCostConfig struct {
	Param      IngressCostParam
	ScaleMicro bool
	MaxMicro   int64
	Policy     IngressCostPolicy
}

type ingressCostConfigJSON struct {
	Param    string `json:"param"`
	Scale    string `json:"scale"`
	MaxMicro int64  `json:"max_micro"`
	Policy   string `json:"policy"`
}

func ParseIngressCostConfigJSON(raw []byte) IngressCostConfig {
	if len(raw) == 0 {
		return IngressCostConfig{}
	}
	var cfg ingressCostConfigJSON
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return IngressCostConfig{}
	}
	out := IngressCostConfig{
		Param:    ingressCostParamFromString(cfg.Param),
		MaxMicro: cfg.MaxMicro,
		Policy:   ingressCostPolicyFromString(cfg.Policy),
	}
	switch strings.ToLower(strings.TrimSpace(cfg.Scale)) {
	case "micro", "micros", "micro_usd":
		out.ScaleMicro = true
	default:
		out.ScaleMicro = false
	}
	return out
}

func ingressCostParamFromString(param string) IngressCostParam {
	switch strings.ToLower(strings.TrimSpace(param)) {
	case "cost":
		return IngressCostParamCost
	case "cpc":
		return IngressCostParamCPC
	case "bid":
		return IngressCostParamBid
	default:
		return IngressCostDisabled
	}
}

func ingressCostPolicyFromString(policy string) IngressCostPolicy {
	switch strings.ToLower(strings.TrimSpace(policy)) {
	case "reject":
		return IngressCostPolicyReject
	default:
		return IngressCostPolicyIgnore
	}
}

func (c IngressCostConfig) Enabled() bool {
	return c.Param != IngressCostDisabled
}

func (c IngressCostConfig) MarshalJSON() ([]byte, error) {
	if !c.Enabled() {
		return []byte("{}"), nil
	}
	scale := "decimal"
	if c.ScaleMicro {
		scale = "micro"
	}
	policy := "ignore"
	if c.Policy == IngressCostPolicyReject {
		policy = "reject"
	}
	param := ""
	switch c.Param {
	case IngressCostParamCost:
		param = "cost"
	case IngressCostParamCPC:
		param = "cpc"
	case IngressCostParamBid:
		param = "bid"
	}
	return json.Marshal(ingressCostConfigJSON{
		Param:    param,
		Scale:    scale,
		MaxMicro: c.MaxMicro,
		Policy:   policy,
	})
}
