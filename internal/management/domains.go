package management

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
		"errors.go", "rbac.go", "ratelimit.go", "permissions.go", "pgerrors.go",
		"auth_principal.go", "ops.go", "mgmt_pg_gate.go", "domains.go",
	}, LogicFiles: []string{"rbac.go", "errors.go"}, TestPrefixes: []string{"service_test", "handler_test", "middleware_test", "workers_test", "rbac_test", "core_domain_test", "domains_test"}},
	{ID: "billing", Prefixes: []string{"handler_billing", "handler_api_balance", "service_customer", "billing_"}, Files: []string{"billing_money.go"}, LogicFiles: []string{"billing_money.go"}, TestPrefixes: []string{"handler_api_balance_test", "handler_billing", "billing_domain_test", "service_customer"}},
	{ID: "campaign", Prefixes: []string{
		"service_campaign", "handler_campaign", "pacing_", "schedule_", "brand_",
		"campaign_", "service_brands", "handler_warm", "service_warm", "drain_",
		"service_pacing",
	}, Files: []string{"delivery_types.go"}, LogicFiles: []string{"campaign_validate.go"}, TestPrefixes: []string{"campaign_domain_test", "campaign_pacing_test", "brand_fcap_test", "handler_campaigns_test", "delivery_test", "pacing_controller_test"}},
	{ID: "outbox", Prefixes: []string{"outbox_"}, TestPrefixes: []string{"outbox_"}},
	{ID: "operation", Prefixes: []string{"operation_", "api_region_ingest"}, Files: []string{"dedup.go"}, TestPrefixes: []string{"operation_", "api_region", "dedup_", "operation_domain_test"}},
	{ID: "recon", Prefixes: []string{"recon_", "global_spend_", "service_recon"}, LogicFiles: []string{"recon_payload.go"}, TestPrefixes: []string{"recon_", "global_spend_", "recon_domain_test"}},
	{ID: "fraud", Prefixes: []string{"service_fraud", "handler_fraud", "blacklist_", "fraud_"}, TestPrefixes: []string{"service_fraud", "handler_fraud", "blacklist_", "fraud_"}},
	{ID: "node", Prefixes: []string{"node_", "service_node_"}, TestPrefixes: []string{"node_", "service_node_", "node_domain_test"}},
	{ID: "rtb", Prefixes: []string{"service_rtb", "handler_rtb", "service_bid", "rtb_"}, TestPrefixes: []string{"service_rtb", "handler_rtb", "service_bid", "rtb_"}},
	{ID: "shard", Prefixes: []string{"shard_", "slot_", "service_slot_", "service_shard", "handler_slot"}, TestPrefixes: []string{"shard_", "slot_", "service_slot", "handler_slot"}},
	{ID: "settlement", Prefixes: []string{"settlement_"}, TestPrefixes: []string{"settlement_"}},
	{ID: "region", Prefixes: []string{"region_"}, TestPrefixes: []string{"region_"}},
	{ID: "quota", Prefixes: []string{"quota_"}, TestPrefixes: []string{"quota_"}},
	{ID: "delivery", Prefixes: []string{"delivery_", "service_delivery", "handler_delivery"}, TestPrefixes: []string{"delivery_", "handler_delivery"}},
	{ID: "supply", Prefixes: []string{"supply_", "service_supply", "handler_supply"}, TestPrefixes: []string{"supply_", "handler_supply"}},
	{ID: "forecast", Prefixes: []string{"service_forecast", "handler_forecast", "forecast_"}, TestPrefixes: []string{"service_forecast", "handler_forecast", "forecast_", "forecast_domain_test"}},
	{ID: "http", Prefixes: []string{"http_"}, TestPrefixes: []string{"http_"}},
	{ID: "ops", Prefixes: []string{"ops_"}, Files: []string{"ops_alerter.go"}, TestPrefixes: []string{"ops_"}},
	{ID: "audit", Prefixes: []string{"audit"}, TestPrefixes: []string{"audit"}},
	{ID: "api", Prefixes: []string{"handler_api", "api_"}, TestPrefixes: []string{"handler_api", "api_"}},
	{ID: "selfserve", Prefixes: []string{"handler_selfserve", "service_selfserve", "selfserve_"}, TestPrefixes: []string{"handler_selfserve", "selfserve_"}},
	{ID: "integration", Files: []string{
		"auth_client.go", "billing_client.go", "payment_client.go",
		"notifier_client.go", "notifier_routing.go", "alertmanager_webhook.go",
	}, TestPrefixes: []string{"auth_client", "billing_client", "payment_client", "notifier_", "alertmanager_"}},
	{ID: "platform", Prefixes: []string{
		"service_system", "service_autoscaling", "service_margin", "service_mab",
		"service_erasure", "service_notifications", "service_consent", "service_audit",
		"handler_consent", "handler_disputes", "handler_notifications", "handler_payment",
		"handler_webhook", "handler_ops", "handler_forecast", "handler_system",
		"handler_errors", "handler_customers", "privacy_", "credit_", "usage_",
		"volume_", "tls_", "nginx_", "tcp_", "udp_", "redis_", "pg_failover",
		"emergency_", "autoscale", "consent_", "edge_", "fault_", "migration_",
		"repository_", "registry_", "platform_", "supply_export",
	}, LogicFiles: []string{"handler_errors.go"}, TestPrefixes: []string{
		"service_system", "service_autoscaling", "handler_errors", "handler_consent",
		"credit_", "consent_", "emergency_", "platform_", "pg_failover",
	}},
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
	case strings.HasPrefix(base, "handler_"):
		return false
	case strings.HasSuffix(base, "_client.go"):
		return false
	case strings.HasSuffix(base, "_worker.go"):
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

func (d Domain) owns(name string) bool {
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
	"handler_slot_map.go":  true,
	"handler_selfserve.go": true,
	"handler_disputes.go":  true,
	"handler_delivery.go":  true,
}
