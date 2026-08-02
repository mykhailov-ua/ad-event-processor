package controlplane

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"strconv"
	"time"
)

// ValidateInitData verifies the signature of raw initData from Telegram using the bot's secret token.
func ValidateInitData(initData string, botToken string, authDateTTL int64) (map[string]string, error) {
	if botToken == "" {
		return nil, errors.New("empty bot token")
	}

	fields, hash, err := parseInitDataFields(initData)
	if err != nil {
		return nil, err
	}
	if hash == "" {
		return nil, errors.New("missing hash in initData")
	}

	var authDate int64
	var authDateOK bool
	for _, f := range fields {
		if f.key == "auth_date" {
			authDate, err = strconv.ParseInt(f.val, 10, 64)
			if err != nil {
				return nil, errors.New("invalid auth_date format")
			}
			authDateOK = true
			break
		}
	}
	if !authDateOK {
		return nil, errors.New("missing auth_date in initData")
	}

	now := time.Now().Unix()
	if now-authDate > authDateTTL {
		return nil, errors.New("telegram authentication data has expired")
	}

	checkBuf := make([]byte, 0, len(initData))
	checkBuf = appendInitDataCheckString(checkBuf, fields)

	mac := hmac.New(sha256.New, []byte("WebAppData"))
	mac.Write([]byte(botToken))
	secretKey := mac.Sum(nil)

	mac2 := hmac.New(sha256.New, secretKey)
	mac2.Write(checkBuf)
	sum := mac2.Sum(nil)

	if len(hash) != 64 {
		return nil, errors.New("hash signature validation failed")
	}
	var got [32]byte
	if _, err := hex.Decode(got[:], []byte(hash)); err != nil {
		return nil, errors.New("hash signature validation failed")
	}
	if subtle.ConstantTimeCompare(sum, got[:]) != 1 {
		return nil, errors.New("hash signature validation failed")
	}

	res := make(map[string]string, len(fields)+2)
	for _, f := range fields {
		res[f.key] = f.val
	}
	res["hash"] = hash
	return res, nil
}
