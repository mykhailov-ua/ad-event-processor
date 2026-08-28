package campaignmacro

import (
	"strings"
)

const maxSubIDs = 30

const (
	previewClickID = "preview-click-id"
	previewUserID  = "preview-user-id"
)

type PreviewRequest struct {
	Sub1    string
	Country string
	ClickID string
	UserID  string
	FBCLID  string
	GCLID   string
	TTCLID  string
}

type Context struct {
	CampaignID string
	ClickID    string
	UserID     string
	Country    string
	Subs       [maxSubIDs]string
	FBCLID     string
	GCLID      string
	TTCLID     string
}

func PreviewContext(campaignID string, req PreviewRequest) Context {
	clickID := strings.TrimSpace(req.ClickID)
	if clickID == "" {
		clickID = previewClickID
	}
	userID := strings.TrimSpace(req.UserID)
	if userID == "" {
		userID = previewUserID
	}
	ctx := Context{
		CampaignID: strings.TrimSpace(campaignID),
		ClickID:    clickID,
		UserID:     userID,
		Country:    strings.TrimSpace(req.Country),
		FBCLID:     strings.TrimSpace(req.FBCLID),
		GCLID:      strings.TrimSpace(req.GCLID),
		TTCLID:     strings.TrimSpace(req.TTCLID),
	}
	if sub1 := strings.TrimSpace(req.Sub1); sub1 != "" {
		ctx.Subs[0] = sub1
	}
	return ctx
}

func Expand(raw string, ctx Context) (string, []string) {
	if raw == "" {
		return "", nil
	}
	var unresolved []string
	out := expandRedirectMacros(raw, ctx, &unresolved)
	out = expandDoubleBrace(out, ctx, &unresolved)
	return out, unresolved
}

func expandDoubleBrace(raw string, ctx Context, unresolved *[]string) string {
	var b strings.Builder
	b.Grow(len(raw))
	i := 0
	for i < len(raw) {
		if i+1 < len(raw) && raw[i] == '{' && raw[i+1] == '{' {
			end := strings.Index(raw[i+2:], "}}")
			if end < 0 {
				b.WriteByte(raw[i])
				i++
				continue
			}
			key := strings.TrimSpace(raw[i+2 : i+2+end])
			token := raw[i : i+2+end+2]
			value, ok := doubleBraceValue(key, ctx)
			if !ok || value == "" {
				*unresolved = append(*unresolved, token)
				b.WriteString(token)
			} else {
				b.WriteString(value)
			}
			i += 2 + end + 2
			continue
		}
		b.WriteByte(raw[i])
		i++
	}
	return b.String()
}

func doubleBraceValue(key string, ctx Context) (string, bool) {
	switch strings.ToLower(key) {
	case "campaign.id", "campaign_id":
		return ctx.CampaignID, true
	case "country":
		return ctx.Country, true
	case "fbclid":
		return ctx.FBCLID, true
	case "gclid":
		return ctx.GCLID, true
	case "ttclid":
		return ctx.TTCLID, true
	case "click_id":
		return ctx.ClickID, true
	case "user_id":
		return ctx.UserID, true
	default:
		if strings.HasPrefix(strings.ToLower(key), "sub") {
			idx, ok := parseSubIndex(key[3:])
			if ok && idx >= 1 && idx <= maxSubIDs {
				return ctx.Subs[idx-1], true
			}
		}
		return "", false
	}
}

func parseSubIndex(raw string) (int, bool) {
	if raw == "" {
		return 0, false
	}
	n := 0
	for i := 0; i < len(raw); i++ {
		c := raw[i]
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + int(c-'0')
		if n > maxSubIDs {
			return 0, false
		}
	}
	if n == 0 {
		return 0, false
	}
	return n, true
}

func expandRedirectMacros(raw string, ctx Context, unresolved *[]string) string {
	var b strings.Builder
	b.Grow(len(raw))
	i := 0
	for i < len(raw) {
		if raw[i] != '{' {
			b.WriteByte(raw[i])
			i++
			continue
		}
		if i+1 < len(raw) && raw[i+1] == '{' {
			b.WriteByte(raw[i])
			i++
			continue
		}
		if i > 0 && raw[i-1] == '{' {
			b.WriteByte(raw[i])
			i++
			continue
		}
		macroID, end := dispatchRedirectMacro(raw, i)
		switch macroID {
		case redirectMacroClickID:
			b.WriteString(ctx.ClickID)
			i = end
		case redirectMacroUserID:
			b.WriteString(ctx.UserID)
			i = end
		default:
			if macroID >= redirectMacroSub1 && macroID < redirectMacroSub1+redirectMacroID(maxSubIDs) {
				sub := ctx.Subs[macroID-redirectMacroSub1]
				if sub == "" {
					*unresolved = append(*unresolved, raw[i:end])
					b.WriteString(raw[i:end])
				} else {
					b.WriteString(sub)
				}
				i = end
				continue
			}
			b.WriteByte(raw[i])
			i++
		}
	}
	return b.String()
}

type redirectMacroID uint8

const (
	redirectMacroNone redirectMacroID = iota
	redirectMacroClickID
	redirectMacroUserID
	redirectMacroSub1
)

const (
	redirectMacroClickLen = 10
	redirectMacroUserLen  = 9
	redirectMacroSubLen   = 6
)

func dispatchRedirectMacro(base string, i int) (redirectMacroID, int) {
	n := len(base)
	if i >= n || base[i] != '{' || i+1 >= n {
		return redirectMacroNone, i
	}
	switch base[i+1] {
	case 'c':
		if i+redirectMacroClickLen <= n &&
			base[i+2] == 'l' && base[i+3] == 'i' && base[i+4] == 'c' && base[i+5] == 'k' &&
			base[i+6] == '_' && base[i+7] == 'i' && base[i+8] == 'd' && base[i+9] == '}' {
			return redirectMacroClickID, i + redirectMacroClickLen
		}
	case 'u':
		if i+redirectMacroUserLen <= n &&
			base[i+2] == 's' && base[i+3] == 'e' && base[i+4] == 'r' &&
			base[i+5] == '_' && base[i+6] == 'i' && base[i+7] == 'd' && base[i+8] == '}' {
			return redirectMacroUserID, i + redirectMacroUserLen
		}
	case 's':
		if i+redirectMacroSubLen <= n && base[i+2] == 'u' && base[i+3] == 'b' && base[i+5] == '}' {
			digit := base[i+4]
			if digit >= '1' && digit <= '9' {
				return redirectMacroID(redirectMacroSub1 + redirectMacroID(digit-'1')), i + redirectMacroSubLen
			}
		}
		if i+redirectMacroSubLen+1 <= n && base[i+2] == 'u' && base[i+3] == 'b' {
			d1, d2 := base[i+4], base[i+5]
			if d1 >= '1' && d1 <= '3' && d2 >= '0' && d2 <= '9' {
				idx := int(d1-'0')*10 + int(d2-'0')
				if idx >= 10 && idx <= maxSubIDs && base[i+6] == '}' {
					return redirectMacroID(redirectMacroSub1 + redirectMacroID(idx-1)), i + redirectMacroSubLen + 1
				}
			}
		}
	}
	return redirectMacroNone, i
}
