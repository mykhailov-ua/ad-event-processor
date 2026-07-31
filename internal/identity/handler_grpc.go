package identity

import (
	"context"
	"time"

	"espx/internal/identity/pb"
	"espx/pkg/coldpath"
)

func (h *Handler) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	customerID, err := parseOptionalCustomerID(req.CustomerId)
	if err != nil {
		return nil, err
	}
	id, err := h.registerUser(ctx, req.Email, req.Password, req.Role, customerID)
	if err != nil {
		return nil, err
	}
	return &pb.RegisterResponse{UserId: id.String()}, nil
}

func (h *Handler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	dto, err := h.loginDTO(ctx, req.Email, req.Password, req.DurationHours)
	if err != nil {
		return nil, err
	}
	return loginDTOToPB(dto), nil
}

func (h *Handler) VerifyToken(ctx context.Context, req *pb.VerifyTokenRequest) (*pb.VerifyTokenResponse, error) {
	user, err := h.service.VerifyToken(ctx, req.AccessToken)
	if err != nil {
		return nil, mapError(err)
	}
	return &pb.VerifyTokenResponse{User: userToPB(user)}, nil
}

func (h *Handler) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	accessToken, refreshToken, err := h.refreshSession(ctx, req.RefreshToken)
	if err != nil {
		return nil, err
	}
	return &pb.RefreshTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (h *Handler) RevokeToken(ctx context.Context, req *pb.RevokeTokenRequest) (*pb.RevokeTokenResponse, error) {
	if err := h.revokeSession(ctx, req.RefreshToken); err != nil {
		return nil, err
	}
	return &pb.RevokeTokenResponse{}, nil
}

func (h *Handler) CreateAPIKey(ctx context.Context, req *pb.CreateAPIKeyRequest) (*pb.CreateAPIKeyResponse, error) {
	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		t := req.ExpiresAt.AsTime()
		expiresAt = &t
	}
	result, err := h.createAPIKey(ctx, req.Name, expiresAt)
	if err != nil {
		return nil, err
	}
	return createAPIKeyResultToPB(result), nil
}

func (h *Handler) VerifyAPIKey(ctx context.Context, req *pb.VerifyAPIKeyRequest) (*pb.VerifyAPIKeyResponse, error) {
	user, err := h.verifyAPIKeyUser(ctx, req.GetApiKey())
	if err != nil {
		return nil, err
	}
	return &pb.VerifyAPIKeyResponse{User: userToPB(user)}, nil
}

func (h *Handler) ListAPIKeys(ctx context.Context, _ *pb.ListAPIKeysRequest) (*pb.ListAPIKeysResponse, error) {
	keys, err := h.listAPIKeys(ctx)
	if err != nil {
		return nil, err
	}
	return &pb.ListAPIKeysResponse{Keys: coldpath.MapSlice(keys, APIKeyToPB)}, nil
}

func (h *Handler) ChangePassword(ctx context.Context, req *pb.ChangePasswordRequest) (*pb.ChangePasswordResponse, error) {
	user, err := h.requireAuthUser(ctx)
	if err != nil {
		return nil, err
	}
	err = h.service.ChangePassword(ctx, uuidFromPg(user.ID), req.OldPassword, req.NewPassword, h.extractClientIP(ctx), h.extractUserAgent(ctx))
	if err != nil {
		return nil, mapError(err)
	}
	return &pb.ChangePasswordResponse{}, nil
}

func (h *Handler) RequestEmailVerification(ctx context.Context, _ *pb.RequestEmailVerificationRequest) (*pb.RequestEmailVerificationResponse, error) {
	user, err := h.requireAuthUser(ctx)
	if err != nil {
		return nil, err
	}
	token, err := h.service.RequestEmailVerification(ctx, uuidFromPg(user.ID))
	if err != nil {
		return nil, mapError(err)
	}
	return &pb.RequestEmailVerificationResponse{VerificationToken: token}, nil
}

func (h *Handler) ConfirmEmailVerification(ctx context.Context, req *pb.ConfirmEmailVerificationRequest) (*pb.ConfirmEmailVerificationResponse, error) {
	_, err := h.service.ConfirmEmailVerification(ctx, req.VerificationToken)
	if err != nil {
		return nil, mapError(err)
	}
	return &pb.ConfirmEmailVerificationResponse{}, nil
}
