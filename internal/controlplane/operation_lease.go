package controlplane

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

func ValidLeaseState(s string) bool {
	switch LeaseState(s) {
	case LeaseStateBooked, LeaseStateExecuting, LeaseStateCompleted, LeaseStateExpired:
		return true
	default:
		return false
	}
}
