package licensing

import (
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"io"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
)

const (
	assetSealVersion byte = 1

	AssetLabelEdge = "edge-bpf"

	AssetLabelUnifiedFilter = "unified-filter"
)

var (
	ErrSealOpen     = errors.New("sealed asset open failed")
	ErrSealFormat   = errors.New("invalid sealed asset format")
	ErrSealTampered = errors.New("sealed asset tampered")
)

func SealAsset(label string, plaintext []byte, mck [32]byte) ([]byte, error) {
	if len(plaintext) == 0 {
		return nil, ErrSealFormat
	}
	key, err := assetAEADKey(mck, label)
	if err != nil {
		return nil, err
	}
	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := aead.Seal(nil, nonce, plaintext, []byte(label))
	out := make([]byte, 0, 1+1+len(label)+len(nonce)+len(ciphertext))
	out = append(out, assetSealVersion, byte(len(label)))
	out = append(out, []byte(label)...)
	out = append(out, nonce...)
	out = append(out, ciphertext...)
	return out, nil
}

func OpenAsset(label string, sealed []byte, mck [32]byte) ([]byte, error) {
	if len(sealed) < 1+1+chacha20poly1305.NonceSize+chacha20poly1305.Overhead {
		return nil, ErrSealFormat
	}
	if sealed[0] != assetSealVersion {
		return nil, ErrSealFormat
	}
	labelLen := int(sealed[1])
	if labelLen <= 0 || labelLen > 64 || 2+labelLen >= len(sealed) {
		return nil, ErrSealFormat
	}
	gotLabel := string(sealed[2 : 2+labelLen])
	if gotLabel != label {
		return nil, ErrSealFormat
	}
	body := sealed[2+labelLen:]
	key, err := assetAEADKey(mck, label)
	if err != nil {
		return nil, err
	}
	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, err
	}
	if len(body) < aead.NonceSize()+chacha20poly1305.Overhead {
		return nil, ErrSealFormat
	}
	nonce := body[:aead.NonceSize()]
	ciphertext := body[aead.NonceSize():]
	plaintext, err := aead.Open(nil, nonce, ciphertext, []byte(label))
	if err != nil {
		return nil, errors.Join(ErrSealOpen, ErrSealTampered, err)
	}
	return plaintext, nil
}

func assetAEADKey(mck [32]byte, label string) ([]byte, error) {
	info := []byte("license-asset-aead-v1:" + label)
	salt := resolveAssetSealSalt()
	out := make([]byte, 32)
	reader := hkdf.New(sha256.New, mck[:], salt, info)
	if _, err := io.ReadFull(reader, out); err != nil {
		return nil, err
	}
	return out, nil
}
