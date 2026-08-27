package postback

import "strings"

// PostbackClickIDWarnings reports missing network click ids required for live CAPI/S2S dispatch.
func PostbackClickIDWarnings(provider string, pb *PostbackPayload) []string {
	if pb == nil {
		pb = &PostbackPayload{}
	}
	provider = strings.ToLower(strings.TrimSpace(provider))
	var out []string
	switch provider {
	case "facebook":
		if strings.TrimSpace(pb.FBCLID) == "" {
			out = append(out, "missing_fbclid: Meta CAPI needs fbclid on the conversion payload or {{fbclid}} on the click URL")
		}
	case "google":
		if strings.TrimSpace(pb.GCLID) == "" {
			out = append(out, "missing_gclid: Google offline conversions need gclid on the conversion payload")
		}
	case "tiktok":
		if strings.TrimSpace(pb.TTCLID) == "" {
			out = append(out, "missing_ttclid: TikTok Events API needs ttclid on the conversion payload")
		}
	case "taboola":
		if strings.TrimSpace(pb.TBLCI) == "" {
			out = append(out, "missing_tblci: Taboola S2S needs tblci (or click_id alias) on the conversion payload")
		}
	case "outbrain":
		if strings.TrimSpace(pb.OBClickID) == "" {
			out = append(out, "missing_ob_click_id: Outbrain S2S needs ob_click_id on the conversion payload")
		}
	case "microsoft_ads":
		if strings.TrimSpace(pb.MSCLKID) == "" {
			out = append(out, "missing_msclkid: Microsoft Ads offline conversions need msclkid on the conversion payload")
		}
	}
	return out
}
