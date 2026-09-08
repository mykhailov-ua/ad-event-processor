package ingest

import (
	"net/http"
	"strconv"

	"ad-event-processor/pkg/antifraudtelemetry"

	"github.com/google/uuid"
	"github.com/panjf2000/gnet/v2"
)

func (h *AdsPacketHandler) reactAntifraudChallenge(req *Request, c gnet.Conn, ctx *ConnContext) gnet.Action {
	startMono := monotonicNano()
	if h == nil || req == nil || len(h.attestationKeys) == 0 {
		h.write(c, respNotFound, ctx)
		h.recordMetrics(startMono, http.StatusNotFound)
		return gnet.None
	}
	campaignID, ok := parseAntifraudChallengeCampaignID(req.Path)
	if !ok {
		h.write(c, respClickBadRequest, ctx)
		h.recordMetrics(startMono, http.StatusBadRequest)
		return gnet.None
	}
	now := antifraudtelemetry.NowUnix()
	token, err := antifraudtelemetry.MintChallenge(
		h.attestationKeys[0].Secret,
		campaignID,
		now,
		antifraudtelemetry.ChallengeTTLDefault(),
		antifraudtelemetry.DefaultPoWDifficulty(),
	)
	if err != nil {
		h.write(c, respInternalError, ctx)
		h.recordMetrics(startMono, http.StatusInternalServerError)
		return gnet.None
	}
	body := appendAntifraudChallengeJSON(ctx.BufSlice[:0], token, antifraudtelemetry.DefaultPoWDifficulty())
	resp := buildAntifraudChallengeGnetResponse(body)
	h.write(c, resp, ctx)
	h.recordMetrics(startMono, http.StatusOK)
	return gnet.None
}

func parseAntifraudChallengeCampaignID(path []byte) (uuid.UUID, bool) {
	if !httpPathHasPrefix(path, antifraudChallengePath) {
		return uuid.Nil, false
	}
	key := []byte("campaign_id=")
	idx := bytesIndex(path, key)
	if idx < 0 {
		return uuid.Nil, false
	}
	start := idx + len(key)
	end := start
	for end < len(path) && path[end] != '&' {
		end++
	}
	if end-start != 36 {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(unsafeString(path[start:end]))
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

func appendAntifraudChallengeJSON(dst []byte, token string, difficulty uint8) []byte {
	dst = append(dst, []byte("{\"challenge_token\":\"")...)
	dst = append(dst, token...)
	dst = append(dst, []byte("\",\"difficulty\":")...)
	dst = strconv.AppendUint(dst, uint64(difficulty), 10)
	dst = append(dst, '}')
	return dst
}

func buildAntifraudChallengeGnetResponse(body []byte) []byte {
	prefix := []byte("HTTP/1.1 200 OK\r\nContent-Type: application/json; charset=utf-8\r\nCache-Control: no-store\r\nAccess-Control-Allow-Origin: *\r\nContent-Length: ")
	suffix := []byte("\r\nConnection: keep-alive\r\n\r\n")
	out := make([]byte, 0, len(prefix)+16+len(suffix)+len(body))
	out = append(out, prefix...)
	out = strconv.AppendInt(out, int64(len(body)), 10)
	out = append(out, suffix...)
	out = append(out, body...)
	return out
}

func bytesIndex(b, sub []byte) int {
	if len(sub) == 0 || len(b) < len(sub) {
		return -1
	}
	for i := 0; i+len(sub) <= len(b); i++ {
		match := true
		for j := range sub {
			if b[i+j] != sub[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}
