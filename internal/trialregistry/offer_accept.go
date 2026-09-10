package trialregistry

import (
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	EnvOfferVersion        = "VENDOR_OFFER_VERSION"
	EnvOfferSummaryURL     = "VENDOR_OFFER_SUMMARY_URL"
	DefaultOfferVersion    = "2026-09-09"
	DefaultOfferSummaryURL = "https://bidshard.com/offer.html"
	AcceptSourceTelegram   = "telegram"
	AcceptSourceWeb        = "web"
	AcceptSourceVendorAPI  = "vendor_api"
)

type OfferAcceptRecord struct {
	TelegramID   string    `json:"telegram_id"`
	OfferVersion string    `json:"offer_version"`
	AcceptedAt   time.Time `json:"accepted_at"`
	Source       string    `json:"source,omitempty"`
}

func OfferSummaryURL() string {
	if u := strings.TrimSpace(os.Getenv(EnvOfferSummaryURL)); u != "" {
		return u
	}
	return DefaultOfferSummaryURL
}

func CurrentOfferVersion() string {
	if v := strings.TrimSpace(os.Getenv(EnvOfferVersion)); v != "" {
		return v
	}
	return DefaultOfferVersion
}

func (r *Registry) AcceptOffer(telegramID, offerVersion, source string) error {
	telegramID = normalizeTelegramID(telegramID)
	if telegramID == "" {
		return fmt.Errorf("telegram_id is required")
	}
	offerVersion = strings.TrimSpace(offerVersion)
	if offerVersion == "" {
		offerVersion = CurrentOfferVersion()
	}
	if offerVersion != CurrentOfferVersion() {
		return ErrOfferVersionMismatch
	}
	source = strings.TrimSpace(source)
	if source == "" {
		source = AcceptSourceTelegram
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	snap, err := r.loadLocked()
	if err != nil {
		return err
	}
	upsertOfferAcceptanceLocked(snap, telegramID, offerVersion, source, time.Now().UTC())
	return r.saveLocked(snap)
}

func (r *Registry) HasOfferAcceptance(telegramID, offerVersion string) (bool, error) {
	telegramID = normalizeTelegramID(telegramID)
	if telegramID == "" {
		return false, fmt.Errorf("telegram_id is required")
	}
	if strings.TrimSpace(offerVersion) == "" {
		offerVersion = CurrentOfferVersion()
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	snap, err := r.loadLocked()
	if err != nil {
		return false, err
	}
	return hasOfferAcceptanceLocked(snap, telegramID, offerVersion), nil
}

func hasOfferAcceptanceLocked(snap *fileSnapshot, telegramID, offerVersion string) bool {
	for i := range snap.OfferAcceptances {
		rec := snap.OfferAcceptances[i]
		if rec.TelegramID != telegramID {
			continue
		}
		if rec.OfferVersion == offerVersion {
			return true
		}
	}
	return false
}

func upsertOfferAcceptanceLocked(snap *fileSnapshot, telegramID, offerVersion, source string, at time.Time) {
	for i := range snap.OfferAcceptances {
		if snap.OfferAcceptances[i].TelegramID != telegramID {
			continue
		}
		snap.OfferAcceptances[i].OfferVersion = offerVersion
		snap.OfferAcceptances[i].AcceptedAt = at
		snap.OfferAcceptances[i].Source = source
		return
	}
	snap.OfferAcceptances = append(snap.OfferAcceptances, OfferAcceptRecord{
		TelegramID:   telegramID,
		OfferVersion: offerVersion,
		AcceptedAt:   at,
		Source:       source,
	})
}
