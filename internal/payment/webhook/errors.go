package webhook

import (
	"errors"

	"github.com/jackc/pgx/v5"
)

var (
	ErrInvalidRequestBody   = errors.New("invalid request body")
	ErrWebhookEventNotFound = errors.New("webhook event not found")
)

func mapNotFound(err, notFound error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return notFound
	}
	return err
}
