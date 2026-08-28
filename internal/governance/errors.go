package governance

import "errors"

var (
	ErrBudgetApprovalRequired  = errors.New("budget approval required")
	ErrBudgetApprovalAutoDenied = errors.New("budget approval auto-denied")
	ErrTeamMemberNotFound      = errors.New("team member not found")
)

func ValidationError(msg string) error {
	return validationError(msg)
}

type validationError string

func (e validationError) Error() string {
	return string(e)
}
