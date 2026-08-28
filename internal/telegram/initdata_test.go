package telegram

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestValidateInitData(t *testing.T) {
	t.Parallel()
	botToken := "123456789:ABCdefGhIJKlmNoPQRsTUVwxyZ"
	now := time.Now().Unix()

	values := url.Values{}
	values.Set("auth_date", strconv.FormatInt(now, 10))
	values.Set("query_id", "AAHdJu1bAAAAAN0i7VvHG9Ta")
	values.Set("user", `{"id":279058305,"first_name":"Vladislav","username":"vkovalenko"}`)

	dataCheckString := "auth_date=" + strconv.FormatInt(now, 10) + "\n" +
		"query_id=AAHdJu1bAAAAAN0i7VvHG9Ta\n" +
		"user=" + `{"id":279058305,"first_name":"Vladislav","username":"vkovalenko"}`

	mac := hmac.New(sha256.New, []byte("WebAppData"))
	mac.Write([]byte(botToken))
	secretKey := mac.Sum(nil)

	mac2 := hmac.New(sha256.New, secretKey)
	mac2.Write([]byte(dataCheckString))
	hash := hex.EncodeToString(mac2.Sum(nil))

	values.Set("hash", hash)
	initDataRaw := values.Encode()

	t.Run("Valid signature", func(t *testing.T) {
		res, err := ValidateInitData(initDataRaw, botToken, 300)
		require.NoError(t, err)
		require.Contains(t, res["user"], "279058305")
	})

	t.Run("Expired auth_date", func(t *testing.T) {
		expiredValues := url.Values{}
		expiredValues.Set("auth_date", strconv.FormatInt(now-400, 10))
		expiredValues.Set("query_id", "AAHdJu1bAAAAAN0i7VvHG9Ta")
		expiredValues.Set("user", `{"id":279058305,"first_name":"Vladislav","username":"vkovalenko"}`)

		expiredCheckString := "auth_date=" + strconv.FormatInt(now-400, 10) + "\n" +
			"query_id=AAHdJu1bAAAAAN0i7VvHG9Ta\n" +
			"user=" + `{"id":279058305,"first_name":"Vladislav","username":"vkovalenko"}`

		macExp := hmac.New(sha256.New, secretKey)
		macExp.Write([]byte(expiredCheckString))
		expiredHash := hex.EncodeToString(macExp.Sum(nil))

		expiredValues.Set("hash", expiredHash)
		expiredInitData := expiredValues.Encode()

		_, err := ValidateInitData(expiredInitData, botToken, 300)
		require.Error(t, err)
		require.Contains(t, err.Error(), "expired")
	})

	t.Run("Invalid signature", func(t *testing.T) {
		invalidValues := values
		invalidValues.Set("hash", "wronghash123")
		_, err := ValidateInitData(invalidValues.Encode(), botToken, 300)
		require.Error(t, err)
		require.Contains(t, err.Error(), "validation failed")
	})
}

func FuzzParseInitData(f *testing.F) {
	f.Add("auth_date=1&hash=abc&user=%7B%7D")
	f.Fuzz(func(t *testing.T, initData string) {
		_, _ = ValidateInitData(initData, "123456789:ABCdefGhIJKlmNoPQRsTUVwxyZ", 300)
	})
}
