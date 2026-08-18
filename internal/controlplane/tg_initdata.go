package controlplane

import (
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"strconv"
	"time"
)

const telegramPubKeyHex = "768285854497e1694f4757c91c36b44f23b28b76008697669d2d0b5037d4f958"

var telegramPubKey ed25519.PublicKey

func init() {
	telegramPubKey, _ = hex.DecodeString(telegramPubKeyHex)
}

func ValidateInitData(initData string, botToken string, authDateTTL int64) (map[string]string, error) {
	if botToken == "" {
		return nil, errors.New("empty bot token")
	}

	fields, hash, err := parseInitDataFields(initData)
	if err != nil {
		return nil, err
	}

	var signature string
	for _, f := range fields {
		if f.key == "signature" {
			signature = f.val
			break
		}
	}

	if hash == "" && signature == "" {
		return nil, errors.New("missing hash or signature in initData")
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

	if signature != "" {
		sigBytes, err := hex.DecodeString(signature)
		if err != nil || len(sigBytes) != ed25519.SignatureSize {
			return nil, errors.New("invalid signature format")
		}
		if !ed25519.Verify(telegramPubKey, checkBuf, sigBytes) {
			return nil, errors.New("signature validation failed")
		}
	} else {
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
	}

	res := make(map[string]string, len(fields)+2)
	for _, f := range fields {
		res[f.key] = f.val
	}
	if hash != "" {
		res["hash"] = hash
	}
	return res, nil
}
