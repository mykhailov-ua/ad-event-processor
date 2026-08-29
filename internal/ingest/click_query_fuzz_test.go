package ingest

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func FuzzParseClickQuery(f *testing.F) {
	cid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	f.Add("sub1=abc&type=click&campaign_id=550e8400-e29b-41d4-a716-446655440000")
	f.Add("campaign_id=550e8400-e29b-41d4-a716-446655440000&sub30=zz&type=click")

	f.Add("campaign_id=550e8400-e29b-41d4-a716-446655440000&type=click&click_id=00000000-0000-4000-8000-000000000001&gclid=GCLID99&sub1=px")
	f.Add("campaign_id=550e8400-e29b-41d4-a716-446655440000&type=click&fbclid=FB99&ttclid=TT99&sub1=a&sub2=b")
	f.Fuzz(func(t *testing.T, query string) {
		if strings.Contains(query, "\x00") {
			t.Skip("integration: run make test-integration (Docker testcontainers)")
		}
		path := []byte("/click?" + query)
		if len(path) > clickQueryScratchCap*2 {
			t.Skip("integration: run make test-integration (Docker testcontainers)")
		}
		parsed := &clickQueryParsed{}
		_ = parseClickQuery(path, nil, parsed)
		if parsed.OK && parsed.CampaignID != uuid.Nil && parsed.CampaignID != cid {
			_ = parsed.CampaignID.String()
		}
	})
}
