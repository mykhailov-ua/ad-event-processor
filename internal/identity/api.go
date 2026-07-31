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
	user, err := a.h.service.VerifyAPIKey(ctx, apiKey)
	if err != nil {
		return AuthUser{}, err
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
	resp, err := a.h.Login(ctx, &pb.LoginRequest{
		Email:         email,
		Password:      password,
		DurationHours: durationHours,
	})
	if err != nil {
		return LoginResult{}, grpcStatusToError(err)
	}
	if resp == nil {
		return LoginResult{}, nil
	}
	return loginResultFromPB(*resp), nil
}

func (a *authAPI) Register(ctx context.Context, adminAPIKey, email, password, role, customerID string) (RegisterResult, error) {
	if a.h.cfg == nil || a.h.cfg.AdminAPIKey == "" {
		return RegisterResult{}, ErrValidation
	}
	if adminAPIKey == "" || adminAPIKey != string(a.h.cfg.AdminAPIKey) {
		return RegisterResult{}, ErrInvalidCredentials
	}
	var cid uuid.UUID
	var err error
	if customerID != "" {
		cid, err = uuid.Parse(customerID)
		if err != nil {
			return RegisterResult{}, ErrValidation
		}
	}
	id, err := a.h.service.Register(ctx, RegisterDTO{
		Email:      email,
		Password:   password,
		Role:       role,
		CustomerID: cid,
	})
	if err != nil {
		return RegisterResult{}, err
	}
	return RegisterResult{UserID: id}, nil
}

func (a *authAPI) RefreshToken(ctx context.Context, refreshToken string) (RefreshResult, error) {
	durationHrs := int32(1)
	if a.h.cfg != nil && a.h.cfg.DefaultTokenDurationHrs > 0 {
		durationHrs = int32(a.h.cfg.DefaultTokenDurationHrs)
	}
	duration := time.Duration(durationHrs) * time.Hour
	accessToken, newRefresh, err := a.h.service.RefreshToken(ctx, refreshToken, duration)
	if err != nil {
		return RefreshResult{}, err
	}
	return RefreshResult{
		AccessToken:  accessToken,
		RefreshToken: newRefresh,
	}, nil
}

func (a *authAPI) RevokeToken(ctx context.Context, refreshToken string) error {
	return a.h.service.RevokeToken(ctx, refreshToken)
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
