package identity

import "strings"

const (
	RoleAdmin   = "A"
	RoleManager = "M"
	RoleUser    = "U"
)

func NormalizeRole(role string) string {
	switch strings.ToUpper(strings.TrimSpace(role)) {
	case "SUPERADMIN", "ADMIN", "SA", "A":
		return RoleAdmin
	case "MANAGER", "M":
		return RoleManager
	case "CUSTOMER", "USER", "C", "U":
		return RoleUser
	default:
		return strings.ToUpper(strings.TrimSpace(role))
	}
}

func ValidateRegisterRole(role string) (string, error) {
	normalized := NormalizeRole(role)
	if normalized == "" {
		return RoleUser, nil
	}
	switch normalized {
	case RoleAdmin, RoleManager, RoleUser:
		return normalized, nil
	default:
		return "", ErrValidation
	}
}
