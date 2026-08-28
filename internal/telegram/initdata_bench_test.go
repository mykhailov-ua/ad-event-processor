package telegram

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strconv"
	"testing"
	"time"
)

func benchInitDataRaw(b *testing.B) string {
	botToken := "123456789:ABCdefGhIJKlmNoPQRsTUVwxyZ"
	now := time.Now().Unix()
	dataCheckString := "auth_date=" + strconv.FormatInt(now, 10) + "\n" +
		"query_id=AAHdJu1bAAAAAN0i7VvHG9Ta\n" +
		"user=" + `{"id":279058305,"first_name":"Vladislav","username":"vkovalenko"}`
	mac := hmac.New(sha256.New, []byte("WebAppData"))
	mac.Write([]byte(botToken))
	secretKey := mac.Sum(nil)
	mac2 := hmac.New(sha256.New, secretKey)
	mac2.Write([]byte(dataCheckString))
	hash := hex.EncodeToString(mac2.Sum(nil))
	values := url.Values{}
	values.Set("auth_date", strconv.FormatInt(now, 10))
	values.Set("query_id", "AAHdJu1bAAAAAN0i7VvHG9Ta")
	values.Set("user", `{"id":279058305,"first_name":"Vladislav","username":"vkovalenko"}`)
	values.Set("hash", hash)
	return values.Encode()
}

func BenchmarkValidateInitData(b *testing.B) {
	raw := benchInitDataRaw(b)
	botToken := "123456789:ABCdefGhIJKlmNoPQRsTUVwxyZ"
	b.ReportAllocs()
	for b.Loop() {
		_, _ = ValidateInitData(raw, botToken, 300)
	}
}
