package controlplane

import (
	"errors"
	"fmt"
	"time"

	db "espx/internal/domain/db"
	"espx/pkg/dedupkey"

	"github.com/google/uuid"
)

var ErrLeaseQuorumNotMet = errors.New("operation lease book quorum not met")

var ErrOpKeyPoolShed = errors.New("operation lease book shed: op key pool over watermark")

type OpKeyPoolGate interface {
	ShouldShed() bool
}

const (
	defaultLeaseQuorum     = 2
	opLeaseJanitorLockKey1 = int32(0x4d36)
)

func QuorumRequired(replicaCount int) int32 {
	if replicaCount <= 1 {
		return 1
	}
	return defaultLeaseQuorum
}

func ScopeWithAttempt(scope dedupkey.Scope, attempt int32) dedupkey.Scope {
	if attempt < 1 {
		attempt = 1
	}
	if attempt == 1 {
		return scope
	}
	offset := int64(attempt)
	if scope.SeqStart == scope.SeqEnd {
		seq := scope.SeqStart*1000 + offset
		scope.SeqStart = seq
		scope.SeqEnd = seq
		return scope
	}
	scope.SeqStart = scope.SeqStart*1000 + offset
	scope.SeqEnd = scope.SeqEnd*1000 + offset
	return scope
}

func DedupScopeForLease(lease db.OperationLease) (dedupkey.Scope, error) {
	base, attempt, err := DecodeLeaseDedupScope(lease.DedupScope)
	if err != nil {
		return dedupkey.Scope{}, err
	}
	if attempt <= 0 {
		attempt = lease.Attempt
	}
	return ScopeWithAttempt(base, attempt), nil
}

func AuthoritativeLeaseView(lease db.OperationLease, executorNodeID string, now time.Time) error {
	switch LeaseState(lease.LeaseState) {
	case LeaseStateExecuting:
	default:
		opID := uuid.UUID(lease.OpID.Bytes)
		return fmt.Errorf("authoritative lease op_id=%s: state=%s", opID, lease.LeaseState)
	}
	if !lease.DeadlineAt.Valid || !lease.DeadlineAt.Time.After(now) {
		opID := uuid.UUID(lease.OpID.Bytes)
		return fmt.Errorf("authoritative lease op_id=%s: expired deadline", opID)
	}
	if !lease.ExecutorNodeID.Valid || lease.ExecutorNodeID.String != executorNodeID {
		opID := uuid.UUID(lease.OpID.Bytes)
		return fmt.Errorf("authoritative lease op_id=%s: executor mismatch", opID)
	}
	return nil
}
