package controlplane

import (
	"path/filepath"
	"strings"
)

type Domain struct {
	ID           string
	Prefixes     []string
	Files        []string
	LogicFiles   []string
	TestPrefixes []string
}

var ManagementDomains = []Domain{
	{ID: "core", Files: []string{
		"service.go", "handler.go", "middleware.go", "workers.go",
		"errors.go", "rbac.go", "ratelimit.go",
		"ops.go", "postgres_gate.go", "domains.go",
		"serve.go", "api_access.go", "meta_enricher.go",
	}, LogicFiles: []string{"rbac.go", "errors.go"}, TestPrefixes: []string{"service_test", "handler_test", "middleware_test", "workers_test", "rbac_test", "core_domain_test", "domains_test", "admin_gone"}},
	{ID: "billing", Prefixes: []string{"handler_billing", "handler_api_balance", "service_customer", "billing_"}, Files: []string{"billing_money.go", "workers.go"}, LogicFiles: []string{"billing_money.go"}, TestPrefixes: []string{"api_balance", "handler_billing", "billing_domain_test", "service_customer", "ledger_invariant"}},
	{ID: "campaign", Prefixes: []string{
		"service_campaign", "handler_campaign", "pacing_", "schedule_", "brand_",
		"campaign_", "service_brands", "handler_warm", "service_warm", "drain_",
		"service_pacing", "vpp_",
	}, Files: []string{"delivery_types.go", "service_campaign.go", "vpp_controller.go", "vpp_pacing.go"}, LogicFiles: []string{"campaign_validate.go"}, TestPrefixes: []string{"campaign_domain_test", "campaign_pacing_test", "brand_fcap_test", "api_campaigns", "delivery_test", "pacing_controller_test", "cohort_", "vpp_"}},
	{ID: "outbox", Prefixes: []string{"outbox_"}, Files: []string{"outbox.go"}, TestPrefixes: []string{"outbox_"}},
	{ID: "operation", Prefixes: []string{"operation_", "api_region_ingest"}, Files: []string{"dedup.go"}, TestPrefixes: []string{"operation_", "api_region", "dedup_", "operation_domain_test"}},
	{ID: "recon", Prefixes: []string{"recon_", "global_spend_", "service_recon"}, Files: []string{"recon.go"}, LogicFiles: []string{"recon.go"}, TestPrefixes: []string{"recon_", "global_spend_", "recon_domain_test"}},
	{ID: "fraud", Prefixes: []string{"service_fraud", "blacklist_", "worker_blacklist_", "fraud_"}, TestPrefixes: []string{"service_fraud", "blacklist_", "fraud_"}},
	{ID: "node", Prefixes: []string{"node_", "service_node_"}, Files: []string{"workers.go", "service_node.go"}, TestPrefixes: []string{"node_", "service_node_", "node_domain_test", "global_region_"}},
	{ID: "rtb", Prefixes: []string{"service_rtb", "service_bid", "rtb_", "floor_optimizer_"}, Files: []string{"workers.go", "budget_delta_consumer.go"}, TestPrefixes: []string{"service_rtb", "api_rtb", "service_bid", "rtb_", "floor_optimizer"}},
	{ID: "shard", Prefixes: []string{"shard_", "slot_", "service_slot_"}, Files: []string{"shard_control.go"}, TestPrefixes: []string{"shard_", "slot_", "service_slot"}},
	{ID: "settlement", Prefixes: []string{"settlement_"}, Files: []string{"settlement.go", "service_gtax.go"}, TestPrefixes: []string{"settlement_", "gtax_"}},
	{ID: "region", Prefixes: []string{"region_"}, TestPrefixes: []string{"region_"}},
	{ID: "quota", Prefixes: []string{"quota_"}, Files: []string{"quota.go"}, TestPrefixes: []string{"quota_"}},
	{ID: "delivery", Prefixes: []string{"delivery_", "service_delivery"}, Files: []string{"service_campaign.go"}, TestPrefixes: []string{"delivery_"}},
	{ID: "supply", Prefixes: []string{"supply_", "service_supply"}, TestPrefixes: []string{"supply_", "api_supply"}},
	{ID: "forecast", Prefixes: []string{"service_forecast", "api_forecast", "forecast_"}, TestPrefixes: []string{"service_forecast", "api_forecast", "forecast_", "forecast_domain_test"}},
	{ID: "http", Prefixes: []string{"http_"}, TestPrefixes: []string{"http_"}},
	{ID: "ops", Prefixes: []string{"ops_"}, Files: []string{"ops_readers.go"}, TestPrefixes: []string{"ops_"}},
	{ID: "audit", Prefixes: []string{"audit"}, TestPrefixes: []string{"audit"}},
	{ID: "platform", Prefixes: []string{
		"service_system", "service_autoscaling", "service_margin", "service_mab",
		"service_erasure", "service_notifications", "service_consent", "service_audit",
		"handler_disputes", "handler_notifications", "handler_webhook", "api_forecast", "handler_system",
		"api_errors", "handler_customers", "privacy_", "credit_", "usage_",
		"volume_", "tls_", "nginx_", "tcp_", "udp_", "redis_", "pg_failover",
		"emergency_", "autoscale", "consent_", "edge_", "fault_", "migration_",
		"repository_", "registry_", "platform_", "supply_export",
	}, Files: []string{
		"policy_init.go", "telemetry_pulse.go", "vendor_telemetry.go",
		"control_fanout.go", "service_platform.go", "workers.go",
	}, LogicFiles: []string{"api_errors.go"}, TestPrefixes: []string{
		"service_platform", "service_system", "service_autoscaling", "api_errors",
		"credit_", "consent_", "emergency_", "platform_", "pg_failover",
		"support_feedback", "events_retention", "system_state",
	}},
	{ID: "api", Prefixes: []string{"api", "api_"}, Files: []string{
		"adminapi_wire.go",
		"dry_run.go",
	}, TestPrefixes: []string{"api", "api_", "dry_run", "support_bundle", "support_feedback"}},
	{ID: "selfserve", Prefixes: []string{"api_selfserve", "service_selfserve", "selfserve_"}, TestPrefixes: []string{"api_selfserve", "selfserve_"}},
	{ID: "integration", Files: []string{
		"client_integration.go", "notifier_routing.go", "alertmanager_webhook.go",
	}, TestPrefixes: []string{"client_integration", "client_auth", "client_billing", "client_payment", "notifier_", "alertmanager_"}},
}

const domainBusinessLogicCoverageMin = 0.80

func FileDomain(name string) string {
	base := filepath.Base(name)
	if strings.HasSuffix(base, "_test.go") {
		return ""
	}
	for _, d := range ManagementDomains {
		if d.owns(base) {
			return d.ID
		}
	}
	return ""
}

func IsBusinessLogicFile(name string) bool {
	base := filepath.Base(name)
	if strings.HasSuffix(base, "_test.go") {
		return false
	}
	switch {
	case base == "handler.go", base == "middleware.go", base == "workers.go":
		return false
	case strings.HasPrefix(base, "handler_"), strings.HasPrefix(base, "api_"):
		return false
	case strings.HasSuffix(base, "_client.go"), strings.HasPrefix(base, "client_"):
		return false
	case strings.HasPrefix(base, "worker_"):
		return false
	default:
		return true
	}
}

func DomainTestFiles(domainID string) []string {
	d := domainByID(domainID)
	if d == nil {
		return nil
	}
	return d.testFileStems()
}

func domainByID(id string) *Domain {
	for i := range ManagementDomains {
		if ManagementDomains[i].ID == id {
			return &ManagementDomains[i]
		}
	}
	return nil
}

func legacyWorkerFileName(name string) (string, bool) {
	if name == "worker_blacklist_janitor.go" {
		return "", false
	}
	if !strings.HasPrefix(name, "worker_") || !strings.HasSuffix(name, ".go") {
		return "", false
	}
	stem := strings.TrimSuffix(strings.TrimPrefix(name, "worker_"), ".go")
	if stem == "privacy" {
		return "privacy_workers.go", true
	}
	return stem + "_worker.go", true
}

func (d Domain) ownsExact(name string) bool {
	for _, f := range d.Files {
		if name == f {
			return true
		}
	}
	for _, p := range d.Prefixes {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

func (d Domain) owns(name string) bool {
	if d.ownsExact(name) {
		return true
	}
	if legacy, ok := legacyWorkerFileName(name); ok {
		return d.ownsExact(legacy)
	}
	return false
}

func (d Domain) hasTestFile(name string) bool {
	if !strings.HasSuffix(name, "_test.go") {
		return false
	}
	stem := strings.TrimSuffix(name, "_test.go")
	for _, p := range d.TestPrefixes {
		if strings.HasPrefix(stem, p) || stem == strings.TrimSuffix(p, "_test") {
			return true
		}
	}
	for _, p := range d.Prefixes {
		if strings.HasPrefix(stem, p) {
			return true
		}
	}
	for _, f := range d.Files {
		if stem == strings.TrimSuffix(f, ".go") {
			return true
		}
	}
	return false
}

func (d Domain) testFileStems() []string {
	out := make([]string, 0, len(d.TestPrefixes)+1)
	out = append(out, d.ID+"_domain_test.go")
	for _, p := range d.TestPrefixes {
		out = append(out, p+"_test.go")
	}
	return out
}

func domainLogicFiles(domainID string) map[string]bool {
	d := domainByID(domainID)
	if d == nil || len(d.LogicFiles) == 0 {
		return nil
	}
	out := make(map[string]bool, len(d.LogicFiles))
	for _, f := range d.LogicFiles {
		out[f] = true
	}
	return out
}

var handlerSQLCAllowlist = map[string]bool{
	"handler.go":           true,
	"handler_selfserve.go": true,
	"handler_disputes.go":  true,
}
