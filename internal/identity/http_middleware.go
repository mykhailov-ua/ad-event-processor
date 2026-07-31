package identity

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/redis/go-redis/v9"
)

type contextKey string

const (
	AuthorizationPayloadKey contextKey = "authorization_payload"
)

const (
	authorizationHeaderKey  = "authorization"
	authorizationTypeBearer = "bearer"
)

func AuthMiddleware(tokenMaker Maker, rdb redis.UniversalClient, allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authorizationHeader := r.Header.Get(authorizationHeaderKey)
			if len(authorizationHeader) == 0 {
				err := errors.New("authorization header is not provided")
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			accessToken, ok := parseBearerToken(authorizationHeader)
			if !ok {
				err := errors.New("invalid authorization header format")
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			payload, err := tokenMaker.VerifyToken(accessToken)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			revoked, errRev := CheckTokenRevocation(r.Context(), rdb, payload)
			if errRev != nil {
				AuthTokenErrors.WithLabelValues("revocation_check_failed").Inc()
				slog.Error("failed to check token revocation in redis (fail-closed)", slog.Any("error", errRev))
				http.Error(w, "authorization check failed", http.StatusUnauthorized)
				return
			}
			if revoked {
				http.Error(w, "token revoked", http.StatusUnauthorized)
				return
			}

			if len(allowedRoles) > 0 {
				authorized := false
				normalizedRole := NormalizeRole(payload.Role)
				for _, role := range allowedRoles {
					if normalizedRole == NormalizeRole(role) {
						authorized = true
						break
					}
				}
				if !authorized {
					http.Error(w, "permission denied", http.StatusForbidden)
					return
				}
			}

			ctx := context.WithValue(r.Context(), AuthorizationPayloadKey, payload)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetPayload(ctx context.Context) (*Payload, error) {
	payload, ok := ctx.Value(AuthorizationPayloadKey).(*Payload)
	if !ok {
		return nil, errors.New("context does not contain authorization payload")
	}
	return payload, nil
}
