package platformadmin

import (
	"context"
	"errors"
	"strings"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/identity"
	identitydb "ad-event-processor/internal/identity/db"
	"ad-event-processor/internal/licensing"
	licverify "ad-event-processor/internal/licensing/verify"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type ActivationHost interface {
	Pool() *pgxpool.Pool
	ApplyLicenseToken(ctx context.Context, token string) error
	ErrValidation(msg string) error
	AuditOwnerActivation(ctx context.Context, deploymentID, customerID, ownerUserID uuid.UUID)
}

type InviteAcceptHost interface {
	Pool() *pgxpool.Pool
	InviteRedis() redis.UniversalClient
	ErrValidation(msg string) error
}

type ActivateOwnerRequest struct {
	LicenseToken string
	Email        string
	Password     string
	TeamName     string
}

type AcceptTeamInviteRequest struct {
	Token    string
	Password string
}

type ActivatedOwner struct {
	Email string
	Role  string
}

func ActivateOwner(ctx context.Context, host ActivationHost, req ActivateOwnerRequest) (ActivatedOwner, error) {
	if host == nil || host.Pool() == nil {
		return ActivatedOwner{}, errPlatformServiceUnavailable()
	}
	req.LicenseToken = strings.TrimSpace(req.LicenseToken)
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.TeamName = strings.TrimSpace(req.TeamName)
	if req.LicenseToken == "" || req.Email == "" || req.Password == "" || req.TeamName == "" {
		return ActivatedOwner{}, host.ErrValidation("license_token, email, password, and team_name are required")
	}
	if err := identity.ValidatePassword(req.Password); err != nil {
		return ActivatedOwner{}, host.ErrValidation(err.Error())
	}

	claims, err := licensing.VerifyJWTResolved(req.LicenseToken)
	if err != nil {
		if errors.Is(err, licverify.ErrInvalidSignature) || errors.Is(err, licverify.ErrInvalidTokenFormat) {
			return ActivatedOwner{}, host.ErrValidation("invalid license token")
		}
		return ActivatedOwner{}, err
	}
	deploymentID, err := uuid.Parse(strings.TrimSpace(claims.DeploymentID))
	if err != nil {
		return ActivatedOwner{}, host.ErrValidation("invalid deployment id in license token")
	}
	if err := licensing.CheckHostActivation(ctx, host.Pool(), claims, licensing.HostFingerprint()); err != nil {
		switch {
		case errors.Is(err, licverify.ErrFingerprintMismatch),
			errors.Is(err, licverify.ErrFingerprintRequired),
			errors.Is(err, licverify.ErrActivationLimit):
			return ActivatedOwner{}, host.ErrValidation(err.Error())
		default:
			return ActivatedOwner{}, err
		}
	}

	pool := host.Pool()
	var claimed bool
	if err := pool.QueryRow(ctx, `SELECT TRUE FROM owner_activations WHERE deployment_id = $1`, deploymentID).Scan(&claimed); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return ActivatedOwner{}, err
	} else if err == nil {
		return ActivatedOwner{}, ErrDeploymentAlreadyClaimed
	}

	hasher, err := identity.NewPasswordHasher(32768, 2, 2)
	if err != nil {
		return ActivatedOwner{}, err
	}
	hash, err := hasher.HashPassword(req.Password)
	if err != nil {
		return ActivatedOwner{}, err
	}

	customerID := uuid.New()
	ownerUserID := uuid.New()
	err = pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		idq := identitydb.New(tx)
		if _, err := q.CreateCustomer(ctx, db.CreateCustomerParams{
			ID:       domain.ToUUID(customerID),
			Name:     req.TeamName,
			Balance:  0,
			Currency: "USD",
		}); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO users (id, email, password_hash, role, customer_id, email_verified)
			VALUES ($1, $2, $3, $4, $5, TRUE)`,
			ownerUserID, req.Email, hash, identityRoleTeamLead, customerID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO owner_activations (deployment_id, customer_id, owner_user_id)
			VALUES ($1, $2, $3)`,
			deploymentID, customerID, ownerUserID); err != nil {
			return err
		}
		return idq.CreatePasswordHistoryEntry(ctx, identitydb.CreatePasswordHistoryEntryParams{
			UserID:       pgtype.UUID{Bytes: ownerUserID, Valid: true},
			PasswordHash: hash,
		})
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ActivatedOwner{}, host.ErrValidation("email already registered")
		}
		return ActivatedOwner{}, err
	}

	if err := host.ApplyLicenseToken(ctx, req.LicenseToken); err != nil {
		return ActivatedOwner{}, err
	}
	host.AuditOwnerActivation(ctx, deploymentID, customerID, ownerUserID)

	return ActivatedOwner{Email: req.Email, Role: identityRoleTeamLead}, nil
}

func AcceptTeamInvite(ctx context.Context, host InviteAcceptHost, req AcceptTeamInviteRequest) (ActivatedOwner, error) {
	if host == nil || host.Pool() == nil {
		return ActivatedOwner{}, errPlatformServiceUnavailable()
	}
	req.Token = strings.TrimSpace(req.Token)
	if req.Token == "" || req.Password == "" {
		return ActivatedOwner{}, host.ErrValidation("token and password are required")
	}
	if err := identity.ValidatePassword(req.Password); err != nil {
		return ActivatedOwner{}, host.ErrValidation(err.Error())
	}

	payload, err := LoadTeamInvite(ctx, host.InviteRedis(), req.Token)
	if err != nil {
		return ActivatedOwner{}, err
	}

	hasher, err := identity.NewPasswordHasher(32768, 2, 2)
	if err != nil {
		return ActivatedOwner{}, err
	}
	hash, err := hasher.HashPassword(req.Password)
	if err != nil {
		return ActivatedOwner{}, err
	}

	pool := host.Pool()
	var email, role string
	err = pool.QueryRow(ctx, `
		SELECT email, role
		FROM users
		WHERE id = $1 AND customer_id = $2`,
		payload.UserID, payload.CustomerID).Scan(&email, &role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ActivatedOwner{}, ErrInviteInvalid
		}
		return ActivatedOwner{}, err
	}
	if !strings.EqualFold(email, payload.Email) {
		return ActivatedOwner{}, ErrInviteInvalid
	}

	err = pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		idq := identitydb.New(tx)
		tag, err := tx.Exec(ctx, `
			UPDATE users
			SET password_hash = $1, email_verified = TRUE, updated_at = NOW()
			WHERE id = $2 AND customer_id = $3`,
			hash, payload.UserID, payload.CustomerID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrInviteInvalid
		}
		return idq.CreatePasswordHistoryEntry(ctx, identitydb.CreatePasswordHistoryEntryParams{
			UserID:       pgtype.UUID{Bytes: payload.UserID, Valid: true},
			PasswordHash: hash,
		})
	})
	if err != nil {
		return ActivatedOwner{}, err
	}
	_ = DeleteTeamInvite(ctx, host.InviteRedis(), req.Token)

	return ActivatedOwner{Email: email, Role: role}, nil
}

const identityRoleTeamLead = "TL"
