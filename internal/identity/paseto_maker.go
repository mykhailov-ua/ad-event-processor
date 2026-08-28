package identity

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/o1egl/paseto/v2"
	"golang.org/x/crypto/chacha20poly1305"
)

type PasetoMaker struct {
	paseto       *paseto.V2
	symmetricKey []byte
}

func NewPasetoMaker(symmetricKey string) (Maker, error) {
	if len(symmetricKey) != chacha20poly1305.KeySize {
		return nil, errors.New("invalid key size")
	}

	maker := &PasetoMaker{
		paseto:       paseto.NewV2(),
		symmetricKey: []byte(symmetricKey),
	}
	return maker, nil
}

func (pm *PasetoMaker) CreateToken(userID, sessionID uuid.UUID, role string, customerID uuid.UUID, duration time.Duration) (string, error) {
	payload, err := NewPayload(userID, sessionID, role, customerID, duration)
	if err != nil {
		return "", err
	}

	return pm.paseto.Encrypt(pm.symmetricKey, payload, nil)
}

func (pm *PasetoMaker) VerifyToken(token string) (*Payload, error) {
	payload := &Payload{}

	err := pm.paseto.Decrypt(token, pm.symmetricKey, payload, nil)
	if err != nil {
		return nil, ErrInvalidToken
	}

	err = payload.Valid()
	if err != nil {
		return nil, err
	}

	return payload, nil
}
