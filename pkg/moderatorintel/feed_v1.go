package moderatorintel

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/netip"
	"strings"
	"time"
)

const FormatV1 = "moderator_intel_v1"

const (
	NetworkMeta   = "meta"
	NetworkGoogle = "google"
	NetworkTikTok = "tiktok"
	NetworkOther  = "other"
)

type Entry struct {
	Prefix  netip.Prefix
	Network uint8
}

type FeedV1 struct {
	Format    string
	Source    string
	ExpiresAt time.Time
	Entries   []Entry
}

type feedV1Wire struct {
	Format    string          `json:"format"`
	Source    string          `json:"source"`
	ExpiresAt string          `json:"expires_at"`
	Entries   []feedEntryWire `json:"entries"`
}

type feedEntryWire struct {
	Prefix  string `json:"prefix"`
	Network string `json:"network"`
}

func NetworkID(name string) (uint8, bool) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case NetworkMeta, "facebook", "fb":
		return 1, true
	case NetworkGoogle, "goog", "google_ads":
		return 2, true
	case NetworkTikTok, "tt":
		return 3, true
	case NetworkOther, "", "unknown":
		return 4, true
	default:
		return 0, false
	}
}

func NetworkName(id uint8) string {
	switch id {
	case 1:
		return NetworkMeta
	case 2:
		return NetworkGoogle
	case 3:
		return NetworkTikTok
	case 4:
		return NetworkOther
	default:
		return NetworkOther
	}
}

func VerifySignature(secret []byte, body []byte, sig string) bool {
	if len(secret) == 0 || len(body) == 0 || sig == "" {
		return false
	}
	sigBytes, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(sig))
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(body)
	want := mac.Sum(nil)
	if len(sigBytes) != len(want) {
		return false
	}
	return subtle.ConstantTimeCompare(sigBytes, want) == 1
}

func ParseFeedV1(body []byte, now time.Time) (FeedV1, error) {
	var wire feedV1Wire
	if err := json.Unmarshal(body, &wire); err != nil {
		return FeedV1{}, fmt.Errorf("decode feed: %w", err)
	}
	if strings.TrimSpace(wire.Format) != FormatV1 {
		return FeedV1{}, fmt.Errorf("unsupported feed format %q", wire.Format)
	}
	expiresAt, err := time.Parse(time.RFC3339, strings.TrimSpace(wire.ExpiresAt))
	if err != nil {
		return FeedV1{}, fmt.Errorf("invalid expires_at: %w", err)
	}
	if !expiresAt.After(now.UTC()) {
		return FeedV1{}, fmt.Errorf("feed expired at %s", expiresAt.UTC().Format(time.RFC3339))
	}
	if len(wire.Entries) == 0 {
		return FeedV1{}, fmt.Errorf("feed has no entries")
	}
	out := FeedV1{
		Format:    FormatV1,
		Source:    strings.TrimSpace(wire.Source),
		ExpiresAt: expiresAt.UTC(),
		Entries:   make([]Entry, 0, len(wire.Entries)),
	}
	for i, row := range wire.Entries {
		prefix, err := netip.ParsePrefix(strings.TrimSpace(row.Prefix))
		if err != nil || !prefix.IsValid() {
			return FeedV1{}, fmt.Errorf("entry %d: invalid prefix", i+1)
		}
		netID, ok := NetworkID(row.Network)
		if !ok {
			return FeedV1{}, fmt.Errorf("entry %d: unknown network %q", i+1, row.Network)
		}
		out.Entries = append(out.Entries, Entry{
			Prefix:  prefix.Masked(),
			Network: netID,
		})
	}
	return out, nil
}
