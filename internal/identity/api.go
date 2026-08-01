package identity

import (
	"context"
	"crypto/subtle"
	"errors"
	"net"
	"time"

	"espx/internal/config"
	"espx/internal/identity/db"
	"espx/pkg/clientip"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

const adminAPIKeyMetadata = "x-admin-api-key"

type Handler struct {
	service *Service
	cfg     *config.Config
}

func NewHandler(service *Service, cfg *config.Config) *Handler {
	return &Handler{
		service: service,
		cfg:     cfg,
	}
}

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
	VerifyToken(ctx context.Context, accessToken string) (AuthUser, error)
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

func (a *authAPI) VerifyAPIKey(ctx context.Context, apiKey string) (AuthUser, error) {
	user, err := a.h.verifyAPIKeyUser(ctx, apiKey)
	if err != nil {
		return AuthUser{}, grpcStatusToError(err)
	}
	return user, nil
}

func (a *authAPI) VerifyToken(ctx context.Context, accessToken string) (AuthUser, error) {
	user, err := a.h.verifyTokenUser(ctx, accessToken)
	if err != nil {
		return AuthUser{}, grpcStatusToError(err)
	}
	return user, nil
}

func (a *authAPI) CreateAPIKey(ctx context.Context, bearerToken, name string) (CreateAPIKeyResult, error) {
	accessToken, ok := parseBearerToken("Bearer " + bearerToken)
	if !ok {
		accessToken, ok = parseBearerToken(bearerToken)
	}
	if !ok {
		return CreateAPIKeyResult{}, ErrInvalidToken
	}
	ctx = metadata.NewIncomingContext(ctx, metadata.Pairs(authorizationHeaderKey, authorizationTypeBearer+" "+accessToken))
	result, err := a.h.createAPIKey(ctx, name, nil)
	if err != nil {
		return CreateAPIKeyResult{}, grpcStatusToError(err)
	}
	return result, nil
}

func (a *authAPI) Login(ctx context.Context, email, password string, durationHours int32) (LoginResult, error) {
	result, err := a.h.login(ctx, email, password, durationHours)
	if err != nil {
		return LoginResult{}, grpcStatusToError(err)
	}
	return result, nil
}

func (a *authAPI) Register(ctx context.Context, adminAPIKey, email, password, role, customerID string) (RegisterResult, error) {
	if a.h.cfg == nil || a.h.cfg.AdminAPIKey == "" {
		return RegisterResult{}, ErrValidation
	}
	cfgKey := string(a.h.cfg.AdminAPIKey)
	if adminAPIKey == "" || subtle.ConstantTimeCompare([]byte(adminAPIKey), []byte(cfgKey)) != 1 {
		return RegisterResult{}, ErrInvalidCredentials
	}
	ctx = metadata.NewIncomingContext(ctx, metadata.Pairs(adminAPIKeyMetadata, adminAPIKey))
	result, err := a.h.register(ctx, email, password, role, customerID)
	if err != nil {
		return RegisterResult{}, grpcStatusToError(err)
	}
	return result, nil
}

func (a *authAPI) RefreshToken(ctx context.Context, refreshToken string) (RefreshResult, error) {
	result, err := a.h.refreshSession(ctx, refreshToken)
	if err != nil {
		return RefreshResult{}, grpcStatusToError(err)
	}
	return result, nil
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

func (h *Handler) loginDuration(durationHours int32) time.Duration {
	duration := time.Duration(durationHours) * time.Hour
	if duration <= 0 {
		duration = time.Duration(h.cfg.DefaultTokenDurationHrs) * time.Hour
	} else if duration > 24*time.Hour {
		duration = 24 * time.Hour
	}
	return duration
}

func (h *Handler) loginDTO(ctx context.Context, email, password string, durationHours int32) (LoginDTO, error) {
	dto, err := h.service.Login(
		ctx,
		email,
		password,
		h.extractUserAgent(ctx),
		h.extractClientIP(ctx),
		h.loginDuration(durationHours),
	)
	if err != nil {
		return LoginDTO{}, mapError(err)
	}
	return dto, nil
}

func (h *Handler) login(ctx context.Context, email, password string, durationHours int32) (LoginResult, error) {
	dto, err := h.loginDTO(ctx, email, password, durationHours)
	if err != nil {
		return LoginResult{}, err
	}
	return loginResultFromDTO(dto), nil
}

func (h *Handler) register(ctx context.Context, email, password, role, customerID string) (RegisterResult, error) {
	cid, err := parseOptionalCustomerID(customerID)
	if err != nil {
		return RegisterResult{}, err
	}
	id, err := h.registerUser(ctx, email, password, role, cid)
	if err != nil {
		return RegisterResult{}, err
	}
	return RegisterResult{UserID: id}, nil
}

func (h *Handler) registerUser(ctx context.Context, email, password, role string, customerID uuid.UUID) (uuid.UUID, error) {
	if err := h.requireAdminKey(ctx); err != nil {
		return uuid.Nil, err
	}
	id, err := h.service.Register(ctx, RegisterDTO{
		Email:      email,
		Password:   password,
		Role:       role,
		CustomerID: customerID,
	})
	if err != nil {
		return uuid.Nil, mapError(err)
	}
	return id, nil
}

func (h *Handler) verifyTokenUser(ctx context.Context, accessToken string) (AuthUser, error) {
	user, err := h.service.VerifyToken(ctx, accessToken)
	if err != nil {
		return AuthUser{}, mapError(err)
	}
	return authUserFromDB(user), nil
}

func (h *Handler) verifyAPIKeyUser(ctx context.Context, apiKey string) (AuthUser, error) {
	user, err := h.service.VerifyAPIKey(ctx, apiKey)
	if err != nil {
		return AuthUser{}, mapError(err)
	}
	return authUserFromDB(user), nil
}

func (h *Handler) createAPIKey(ctx context.Context, name string, expiresAt *time.Time) (CreateAPIKeyResult, error) {
	user, err := h.requireAuthUser(ctx)
	if err != nil {
		return CreateAPIKeyResult{}, err
	}
	result, err := h.service.CreateAPIKey(ctx, uuidFromPg(user.ID), name, expiresAt)
	if err != nil {
		return CreateAPIKeyResult{}, mapError(err)
	}
	return result, nil
}

func (h *Handler) refreshSession(ctx context.Context, refreshToken string) (RefreshResult, error) {
	duration := time.Duration(h.cfg.DefaultTokenDurationHrs) * time.Hour
	accessToken, newRefresh, err := h.service.RefreshToken(ctx, refreshToken, duration)
	if err != nil {
		return RefreshResult{}, mapError(err)
	}
	return RefreshResult{
		AccessToken:  accessToken,
		RefreshToken: newRefresh,
	}, nil
}

func (h *Handler) revokeSession(ctx context.Context, refreshToken string) error {
	if err := h.service.RevokeToken(ctx, refreshToken); err != nil {
		return mapError(err)
	}
	return nil
}

func parseOptionalCustomerID(raw string) (uuid.UUID, error) {
	if raw == "" {
		return uuid.UUID{}, nil
	}
	customerID, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, status.Error(codes.InvalidArgument, "invalid customer id")
	}
	return customerID, nil
}

func (h *Handler) extractClientIP(ctx context.Context) string {
	if r, ok := httpRequestFromContext(ctx); ok {
		return clientip.FromRequest(r, clientip.ParseTrusted(h.cfg.TrustedProxies))
	}

	peerIP := "unknown"
	if p, ok := peer.FromContext(ctx); ok {
		host, _, err := net.SplitHostPort(p.Addr.String())
		if err == nil {
			peerIP = host
		} else {
			peerIP = p.Addr.String()
		}
	}

	trusted := clientip.ParseTrusted(h.cfg.TrustedProxies)
	if peerIP == "127.0.0.1" || peerIP == "::1" || peerIP == "bufconn" {
		trusted = clientip.ParseTrusted(append(h.cfg.TrustedProxies, peerIP))
	}

	var xff, xReal string
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if vals := md.Get("x-real-ip"); len(vals) > 0 {
			xReal = vals[0]
		}
		if vals := md.Get("x-forwarded-for"); len(vals) > 0 {
			xff = vals[0]
		}
	}
	return clientip.FromProxyPeer(peerIP, xff, xReal, trusted)
}

func (h *Handler) extractUserAgent(ctx context.Context) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if ua := md.Get("user-agent"); len(ua) > 0 {
			return ua[0]
		}
	}
	return "grpc-client"
}

func (h *Handler) requireAdminKey(ctx context.Context) error {
	if h.cfg == nil || h.cfg.AdminAPIKey == "" {
		return status.Error(codes.PermissionDenied, "admin credentials not configured")
	}
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, "missing credentials")
	}
	keys := md.Get(adminAPIKeyMetadata)
	if len(keys) == 0 || keys[0] == "" {
		return status.Error(codes.PermissionDenied, "admin credentials required")
	}
	cfgKey := string(h.cfg.AdminAPIKey)
	if subtle.ConstantTimeCompare([]byte(keys[0]), []byte(cfgKey)) != 1 {
		return status.Error(codes.PermissionDenied, "admin credentials required")
	}
	return nil
}

func (h *Handler) requireAuthUser(ctx context.Context) (db.User, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return db.User{}, status.Error(codes.Unauthenticated, "missing authorization metadata")
	}
	values := md.Get(authorizationHeaderKey)
	if len(values) == 0 {
		return db.User{}, status.Error(codes.Unauthenticated, "authorization header is not provided")
	}
	accessToken, ok := parseBearerToken(values[0])
	if !ok {
		return db.User{}, status.Error(codes.Unauthenticated, "invalid authorization header format")
	}
	user, err := h.service.VerifyToken(ctx, accessToken)
	if err != nil {
		return db.User{}, mapError(err)
	}
	return user, nil
}

func mapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrRateLimitExceeded) {
		return status.Error(codes.ResourceExhausted, err.Error())
	}
	if errors.Is(err, ErrInvalidCredentials) || errors.Is(err, ErrInvalidToken) || errors.Is(err, ErrExpiredToken) || errors.Is(err, ErrAccountLocked) || errors.Is(err, ErrSessionBlocked) || errors.Is(err, ErrEmailNotVerified) || errors.Is(err, ErrInvalidAPIKey) {
		return status.Error(codes.Unauthenticated, err.Error())
	}
	if errors.Is(err, ErrUserAlreadyExists) {
		return status.Error(codes.AlreadyExists, err.Error())
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return status.Error(codes.NotFound, "user not found")
	}
	if errors.Is(err, ErrValidation) || errors.Is(err, ErrPasswordReuse) {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	return status.Error(codes.Internal, "internal server error")
}
