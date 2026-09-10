package verify

import (
	"crypto/sha256"
	"io"

	"golang.org/x/crypto/hkdf"
)

const DefaultRedisScriptNonceInfoLabel = "license-redis-script-nonce-v1"

func RedisScriptNonceFromMCKWork(mckWork [32]byte) [8]byte {
	var out [8]byte
	reader := hkdf.New(sha256.New, mckWork[:], nil, []byte(DefaultRedisScriptNonceInfoLabel))
	_, _ = io.ReadFull(reader, out[:])
	return out
}
