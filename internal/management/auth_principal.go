package management

import (
	"github.com/google/uuid"
)

var adminAPIKeyNamespace = uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")

func apiKeyPrincipalID(apiKey string) uuid.UUID {
	return uuid.NewSHA1(adminAPIKeyNamespace, []byte(apiKey))
}
