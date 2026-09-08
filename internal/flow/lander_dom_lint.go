package flow

import (
	"context"
	"fmt"

	"ad-event-processor/internal/metrics"
	"ad-event-processor/pkg/landerhost"

	"github.com/google/uuid"
)

func lintHostedVersion(_ context.Context, host HostedLanderHost, landerID uuid.UUID, version int) error {
	if host == nil {
		return fmt.Errorf("hosted lander host unavailable")
	}
	st := host.HostedLanderStore()
	if st == nil {
		return fmt.Errorf("hosted lander store is not configured")
	}
	result, err := st.LintVersion(landerID, version)
	if err != nil {
		return err
	}
	if result.OK() {
		return nil
	}
	for _, v := range result.Violations {
		metrics.LanderDomLintRejectTotal.WithLabelValues(v.Rule).Inc()
	}
	return &landerhost.DomLintError{Result: result}
}
