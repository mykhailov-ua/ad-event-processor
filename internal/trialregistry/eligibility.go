package trialregistry

import (
	"fmt"
	"strings"
	"time"
)

func checkPilotEligible(now time.Time, cooldown time.Duration, anchors []AnchorRecord, in CheckInput) error {
	if err := checkIdentityAnchor(anchors, AnchorTelegram, in.TelegramID, ErrTrialTelegramUsed); err != nil {
		return err
	}
	if err := checkIdentityAnchor(anchors, AnchorUSDTTx, in.USDTTx, ErrTrialWalletUsed); err != nil {
		return err
	}
	if err := checkHWIDAnchor(now, cooldown, anchors, in.HWID); err != nil {
		return err
	}
	return nil
}

func checkIdentityAnchor(anchors []AnchorRecord, anchorType AnchorType, value string, sentinel error) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	for i := range anchors {
		rec := &anchors[i]
		if rec.AnchorType != anchorType || rec.AnchorValue != value {
			continue
		}
		if identityBlocksPilot(rec.Status) {
			return fmt.Errorf("%w: %s", sentinel, value)
		}
	}
	return nil
}

func identityBlocksPilot(status Status) bool {
	switch status {
	case StatusActive, StatusExpired, StatusConverted:
		return true
	default:
		return false
	}
}

func checkHWIDAnchor(now time.Time, cooldown time.Duration, anchors []AnchorRecord, hwid string) error {
	hwid = strings.TrimSpace(hwid)
	if hwid == "" {
		return nil
	}
	cutoff := now.Add(-cooldown)
	for i := range anchors {
		rec := &anchors[i]
		if rec.AnchorType != AnchorHWID || rec.AnchorValue != hwid {
			continue
		}
		if rec.Status != StatusActive && rec.Status != StatusExpired {
			continue
		}
		if rec.IssuedAt.After(cutoff) {
			return fmt.Errorf("%w: %s", ErrTrialHWIDUsed, hwid)
		}
	}
	return nil
}
