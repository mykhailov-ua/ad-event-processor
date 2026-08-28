package authz

import (
	"context"
	"strings"

	"github.com/google/uuid"
)

type userContextKey struct{}

var authenticatedUserKey userContextKey

const (
	RoleAdmin      = "A"
	RoleManager    = "M"
	RoleUser       = "U"
	RoleBuyer      = "B"
	RoleSupport    = "S"
	RoleTeamLead   = "TL"
	RoleMediaBuyer = "MB"
	RolePublisher  = "P"
)

type AuthenticatedUser struct {
	UserID       uuid.UUID
	Role         string
	CustomerID   uuid.UUID
	AuthSource   string
	Scope        Scope
	APIKeyScopes []string
}

func (u AuthenticatedUser) IsUser() bool {
	return u.Role == RoleUser
}

func (u AuthenticatedUser) IsBuyer() bool {
	return u.Role == RoleBuyer
}

func (u AuthenticatedUser) IsTeamLead() bool {
	return NormalizeRole(u.Role) == RoleTeamLead
}

func (u AuthenticatedUser) IsMediaBuyer() bool {
	return NormalizeRole(u.Role) == RoleMediaBuyer
}

func (u AuthenticatedUser) IsPublisher() bool {
	return NormalizeRole(u.Role) == RolePublisher
}

func (u AuthenticatedUser) HasBoundCustomer() bool {
	return u.IsUser() || u.IsBuyer() || u.IsTeamLead() || u.IsMediaBuyer() || u.IsPublisher()
}

func NormalizeRole(role string) string {
	switch strings.ToUpper(strings.TrimSpace(role)) {
	case RoleAdmin, "ADMIN":
		return RoleAdmin
	case RoleManager, "MANAGER":
		return RoleManager
	case RoleUser, "USER":
		return RoleUser
	case RoleBuyer, "BUYER":
		return RoleBuyer
	case RoleSupport, "SUPPORT":
		return RoleSupport
	case RoleTeamLead, "TEAM_LEAD", "TEAMLEAD":
		return RoleTeamLead
	case RoleMediaBuyer, "MEDIA_BUYER", "MEDIABUYER":
		return RoleMediaBuyer
	case RolePublisher, "PUBLISHER":
		return RolePublisher
	default:
		return strings.ToUpper(strings.TrimSpace(role))
	}
}

func WithAuthenticatedUser(ctx context.Context, user AuthenticatedUser) context.Context {
	return context.WithValue(ctx, authenticatedUserKey, user)
}

func GetUser(ctx context.Context) (AuthenticatedUser, bool) {
	u, ok := ctx.Value(authenticatedUserKey).(AuthenticatedUser)
	return u, ok
}
