package management

// LeaseState is the Postgres lease_state enum (§0, migration §10.3).
type LeaseState string

const (
	LeaseStateBooked    LeaseState = "booked"
	LeaseStateExecuting LeaseState = "executing"
	LeaseStateCompleted LeaseState = "completed"
	LeaseStateExpired   LeaseState = "expired"
)

func (s LeaseState) String() string {
	return string(s)
}

// ValidLeaseState reports whether s is a known lease_state value.
func ValidLeaseState(s string) bool {
	switch LeaseState(s) {
	case LeaseStateBooked, LeaseStateExecuting, LeaseStateCompleted, LeaseStateExpired:
		return true
	default:
		return false
	}
}
