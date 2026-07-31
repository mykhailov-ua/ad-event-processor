package identity

import (
	"context"
	"errors"
	"time"

	"espx/internal/identity/db"
	"espx/internal/identity/pb"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthUser struct {
	ID         uuid.UUID
	Email      string
	Role       string
	CustomerID uuid.UUID
}

type LoginResult struct {
	AccessToken  string
	RefreshToken string
	User         AuthUser
}

type RegisterResult struct {
	UserID uuid.UUID
}

type RefreshResult struct {
	AccessToken  string
	RefreshToken string
}

type CreateAPIKeyResult struct {
	ID        string
	Name      string
	RawKey    string
	ExpiresAt *time.Time
}

type AuthAPI interface {
	VerifyAPIKey(ctx context.Context, apiKey string) (AuthUser, error)
	CreateAPIKey(ctx context.Context, bearerToken, name string) (CreateAPIKeyResult, error)
	Login(ctx context.Context, email, password string, durationHours int32) (LoginResult, error)
	Register(ctx context.Context, adminAPIKey, email, password, role, customerID string) (RegisterResult, error)
	RefreshToken(ctx context.Context, refreshToken string) (RefreshResult, error)
	RevokeToken(ctx context.Context, refreshToken string) error
}

type authAPI struct {
	h *Handler
}

func (h *Handler) API() AuthAPI {
	if h == nil {
		return nil
	}
	return &authAPI{h: h}
}

func authUserFromDB(user db.User) AuthUser {
	return AuthUser{
		ID:         uuidFromPg(user.ID),
		Email:      user.Email,
		Role:       user.Role,
		CustomerID: uuidFromPg(user.CustomerID),
	}
}

func loginResultFromPB(resp pb.LoginResponse) LoginResult {
	out := LoginResult{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
	}
	if resp.User != nil {
		uid, _ := uuid.Parse(resp.User.Id)
		cid, _ := uuid.Parse(resp.User.CustomerId)
		out.User = AuthUser{
			ID:         uid,
			Email:      resp.User.Email,
			Role:       resp.User.Role,
			CustomerID: cid,
		}
	}
	return out
}

func (a *authAPI) VerifyAPIKey(ctx context.Context, apiKey string) (AuthUser, error) {
	user, err := a.h.verifyAPIKeyUser(ctx, apiKey)
	if err != nil {
		return AuthUser{}, grpcStatusToError(err)
	}
	return authUserFromDB(user), nil
}

func (a *authAPI) CreateAPIKey(ctx context.Context, bearerToken, name string) (CreateAPIKeyResult, error) {
	accessToken, ok := parseBearerToken("Bearer " + bearerToken)
	if !ok {
		accessToken, ok = parseBearerToken(bearerToken)
	}
	if !ok {
		return CreateAPIKeyResult{}, ErrInvalidToken
	}
	user, err := a.h.service.VerifyToken(ctx, accessToken)
	if err != nil {
		return CreateAPIKeyResult{}, err
	}
	id, rawKey, err := a.h.service.CreateAPIKey(ctx, uuidFromPg(user.ID), name, nil)
	if err != nil {
		return CreateAPIKeyResult{}, err
	}
	return CreateAPIKeyResult{
		ID:     id.String(),
		Name:   name,
		RawKey: rawKey,
	}, nil
}

func (a *authAPI) Login(ctx context.Context, email, password string, durationHours int32) (LoginResult, error) {
	resp, err := a.h.login(ctx, email, password, durationHours)
	if err != nil {
		return LoginResult{}, grpcStatusToError(err)
	}
	return loginResultFromPB(resp), nil
}

func (a *authAPI) Register(ctx context.Context, adminAPIKey, email, password, role, customerID string) (RegisterResult, error) {
	if a.h.cfg == nil || a.h.cfg.AdminAPIKey == "" {
		return RegisterResult{}, ErrValidation
	}
	if adminAPIKey == "" || adminAPIKey != string(a.h.cfg.AdminAPIKey) {
		return RegisterResult{}, ErrInvalidCredentials
	}
	cid, err := parseOptionalCustomerID(customerID)
	if err != nil {
		return RegisterResult{}, grpcStatusToError(err)
	}
	id, err := a.h.registerUser(ctx, email, password, role, cid)
	if err != nil {
		return RegisterResult{}, grpcStatusToError(err)
	}
	return RegisterResult{UserID: id}, nil
}

func (a *authAPI) RefreshToken(ctx context.Context, refreshToken string) (RefreshResult, error) {
	accessToken, newRefresh, err := a.h.refreshSession(ctx, refreshToken)
	if err != nil {
		return RefreshResult{}, grpcStatusToError(err)
	}
	return RefreshResult{
		AccessToken:  accessToken,
		RefreshToken: newRefresh,
	}, nil
}

func (a *authAPI) RevokeToken(ctx context.Context, refreshToken string) error {
	if err := a.h.revokeSession(ctx, refreshToken); err != nil {
		return grpcStatusToError(err)
	}
	return nil
}

func grpcStatusToError(err error) error {
	if err == nil {
		return nil
	}
	st, ok := status.FromError(err)
	if !ok {
		return err
	}
	msg := st.Message()
	switch {
	case errors.Is(err, ErrInvalidCredentials), msg == ErrInvalidCredentials.Error():
		return ErrInvalidCredentials
	case errors.Is(err, ErrInvalidToken), msg == ErrInvalidToken.Error():
		return ErrInvalidToken
	case errors.Is(err, ErrExpiredToken), msg == ErrExpiredToken.Error():
		return ErrExpiredToken
	case errors.Is(err, ErrAccountLocked), msg == ErrAccountLocked.Error():
		return ErrAccountLocked
	case errors.Is(err, ErrSessionBlocked), msg == ErrSessionBlocked.Error():
		return ErrSessionBlocked
	case errors.Is(err, ErrEmailNotVerified), msg == ErrEmailNotVerified.Error():
		return ErrEmailNotVerified
	case errors.Is(err, ErrInvalidAPIKey), msg == ErrInvalidAPIKey.Error():
		return ErrInvalidAPIKey
	case errors.Is(err, ErrRateLimitExceeded), msg == ErrRateLimitExceeded.Error():
		return ErrRateLimitExceeded
	case errors.Is(err, ErrValidation), msg == ErrValidation.Error():
		return ErrValidation
	}
	switch st.Code() {
	case codes.Unauthenticated:
		return ErrInvalidCredentials
	case codes.ResourceExhausted:
		return ErrRateLimitExceeded
	case codes.InvalidArgument:
		return ErrValidation
	}
	return err
}
