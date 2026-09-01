package controlplane

import (
	"context"
	"net/http"
	"strings"
	"time"

	ctrlhttp "ad-event-processor/internal/control/http"
	"ad-event-processor/internal/platformadmin"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func (s *Service) VendorTelemetryInterval() time.Duration {
	if s == nil || s.cfg == nil {
		return 0
	}
	return time.Duration(s.cfg.VendorTelemetryIntervalSec) * time.Second
}

func (s *Service) VendorTelemetryTimeout() time.Duration {
	if s == nil || s.cfg == nil {
		return 0
	}
	return time.Duration(s.cfg.VendorTelemetryTimeoutSec) * time.Second
}

func (s *Service) GeoIPDBPath() string {
	if s == nil || s.cfg == nil {
		return ""
	}
	return s.cfg.GeoIP.DBPath
}

func (s *Service) StripeSecretKey() string {
	if s == nil || s.cfg == nil {
		return ""
	}
	return string(s.cfg.StripeSecretKey)
}

func (s *Service) TelegramBotToken() string {
	if s == nil || s.cfg == nil {
		return ""
	}
	return string(s.cfg.Notifier.TelegramBotToken)
}

func (s *Service) SMTPHost() string {
	if s == nil || s.cfg == nil {
		return ""
	}
	return s.cfg.Notifier.SMTPHost
}

func (s *Service) SMTPPort() string {
	if s == nil || s.cfg == nil {
		return ""
	}
	return s.cfg.Notifier.SMTPPort
}

func (s *Service) StartWorker(fn func()) {
	s.StartBackgroundWorker(fn)
}

func (s *Service) WorkerContext() context.Context {
	if s == nil || s.ctx == nil {
		return context.Background()
	}
	return s.ctx
}

func (s *Service) StartVendorTelemetryWorker(ctx context.Context) {
	platformadmin.StartVendorTelemetryWorker(ctx, s)
}

func (s *Service) TelemetryOptIn() bool {
	return s != nil && s.cfg != nil && s.cfg.TelemetryOptIn
}

func (s *Service) TelemetryURL() string {
	if s == nil || s.cfg == nil {
		return ""
	}
	return string(s.cfg.TelemetryURL)
}

func (s *Service) TelemetryInterval() time.Duration {
	if s == nil || s.cfg == nil {
		return 0
	}
	return time.Duration(s.cfg.TelemetryIntervalSec) * time.Second
}

func (s *Service) TelemetryHTTPTimeout() time.Duration {
	if s == nil || s.cfg == nil {
		return 0
	}
	return time.Duration(s.cfg.TelemetryHTTPTimeoutSec) * time.Second
}

var (
	_ platformadmin.GovernanceHost     = (*Service)(nil)
	_ platformadmin.BudgetApprovalHost = (*Service)(nil)
	_ platformadmin.TeamGovernance     = (*Service)(nil)
	_ platformadmin.ActivationHost     = (*Service)(nil)
	_ platformadmin.InviteAcceptHost   = (*Service)(nil)
)

func (s *Service) teamGovernance() *platformadmin.Governance {
	return platformadmin.NewGovernance(s)
}

func (s *Service) NormalizeTeamRole(role string) (string, error) {
	switch ctrlhttp.NormalizeRole(role) {
	case ctrlhttp.RoleTeamLead, ctrlhttp.RoleMediaBuyer, ctrlhttp.RoleBuyer:
		return ctrlhttp.NormalizeRole(role), nil
	default:
		return "", errValidation("role must be TL, MB, or B")
	}
}

func (s *Service) ErrCustomerNotFound() error { return ErrCustomerNotFound }

func (s *Service) ErrTeamMemberNotFound() error { return ErrTeamMemberNotFound }

func (s *Service) ErrCampaignNotFound() error { return ErrCampaignNotFound }

func (s *Service) ErrBudgetApprovalRequired() error { return ErrBudgetApprovalRequired }

func (s *Service) ErrBudgetApprovalAutoDenied() error { return ErrBudgetApprovalAutoDenied }

func (s *Service) MapCustomerNotFound(err error) error {
	return mapNotFound(err, ErrCustomerNotFound)
}

func (s *Service) MapCampaignNotFound(err error) error {
	return mapNotFound(err, ErrCampaignNotFound)
}

func (s *Service) InviteTeamMember(ctx context.Context, customerID uuid.UUID, email, role string) (platformadmin.TeamMemberDTO, error) {
	return s.teamGovernance().InviteTeamMember(ctx, customerID, email, role)
}

func (s *Service) UpdateTeamMember(ctx context.Context, customerID, userID uuid.UUID, in platformadmin.UpdateTeamMemberRequest) (platformadmin.TeamMemberDTO, error) {
	return s.teamGovernance().UpdateTeamMember(ctx, customerID, userID, in)
}

func (s *Service) ListTeamBudgetApprovals(ctx context.Context, customerID uuid.UUID, limit, offset int) ([]platformadmin.TeamBudgetApprovalDTO, int64, error) {
	return s.teamGovernance().ListTeamBudgetApprovals(ctx, customerID, limit, offset)
}

func (s *Service) ResolveTeamBudgetApproval(ctx context.Context, customerID, approvalID, resolverID uuid.UUID, approve bool) error {
	return s.teamGovernance().ResolveTeamBudgetApproval(ctx, customerID, approvalID, resolverID, approve)
}

func (s *Service) AssignCampaignOwner(ctx context.Context, campaignID, ownerUserID uuid.UUID) error {
	return platformadmin.AssignCampaignOwner(ctx, s, campaignID, ownerUserID)
}

func (s *Service) InviteRedis() redis.UniversalClient {
	if s == nil || len(s.redisShards) == 0 {
		return nil
	}
	return s.redisShards[0]
}

func (s *Service) PublicPanelBaseURL(r *http.Request) string {
	if s != nil && s.cfg != nil {
		if base := strings.TrimSpace(s.cfg.AdminPublicURL); base != "" {
			return strings.TrimRight(base, "/")
		}
		if base := strings.TrimSpace(s.cfg.ManagementURL); base != "" {
			return strings.TrimRight(base, "/")
		}
	}
	if r != nil && r.Host != "" {
		scheme := "https"
		if r.TLS == nil && !strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
			if s == nil || s.cfg == nil || s.cfg.Env == "development" {
				scheme = "http"
			}
		}
		return scheme + "://" + r.Host
	}
	return ""
}
