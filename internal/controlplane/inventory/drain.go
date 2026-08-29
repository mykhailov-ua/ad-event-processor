package inventory

type DrainRow struct {
	File    string
	Role    string
	Domain  string
	Bridge  string
	Drained bool
}

var CompositionDrain = []DrainRow{
	{File: "adminapi_wire.go", Role: "route_registry_wire", Domain: "shell", Drained: true},
	{File: "register.go", Role: "route_catalog", Domain: "shell", Drained: true},
	{File: "service.go", Role: "deps_lazy_inits", Domain: "shell", Drained: true},
	{File: "service_delegates.go", Role: "postgres_redis_audit_gate", Domain: "shell", Drained: true},
	{File: "workers.go", Role: "background_workers", Domain: "shell", Drained: true},
	{File: "serve.go", Role: "http_server", Domain: "shell", Drained: true},
	{File: "handler.go", Role: "top_level_http", Domain: "shell", Drained: true},
	{File: "middleware.go", Role: "auth_middleware", Domain: "shell", Drained: true},
	{File: "license.go", Role: "license_watcher_errors", Domain: "licensingadmin", Bridge: "licensingadmin_bridge.go", Drained: true},
	{File: "admin_static.go", Role: "spa_bootstrap", Domain: "shell", Drained: true},
	{File: "doc.go", Role: "package_boundary", Domain: "shell", Drained: true},
	{File: "shard_wires.go", Role: "postgres_gate_failover", Domain: "shardadmin", Drained: true},
	{File: "fraudadmin_bridge.go", Role: "fraud_wiring", Domain: "fraudadmin", Bridge: "fraudadmin_bridge.go", Drained: true},
	{File: "brand_bridge.go", Role: "brand_host_wiring", Domain: "brand", Bridge: "brand_bridge.go", Drained: true},
	{File: "marginguard_bridge.go", Role: "margin_guard_wiring", Domain: "marginguard", Bridge: "marginguard_bridge.go", Drained: true},
	{File: "licensingadmin_bridge.go", Role: "licensing_wiring", Domain: "licensingadmin", Bridge: "licensingadmin_bridge.go", Drained: true},
	{File: "settingsadmin_bridge.go", Role: "settings_smartalerts_wiring", Domain: "settingsadmin,smartalerts", Bridge: "settingsadmin_bridge.go", Drained: true},
	{File: "service.go", Role: "automation_invite_ops_wiring", Domain: "automation,platformadmin", Bridge: "service.go", Drained: true},
	{File: "billing_bridge.go", Role: "billing_crypto_wiring", Domain: "billingadmin,payment", Bridge: "billing_bridge.go", Drained: true},
	{File: "campaign_autoscale_bridge.go", Role: "autoscale_wiring", Domain: "campaign", Bridge: "campaign_autoscale_bridge.go", Drained: true},
	{File: "campaign_bandit_bridge.go", Role: "delivery_bandit_worker_wiring", Domain: "campaign,flow", Bridge: "campaign_bandit_bridge.go", Drained: true},
	{File: "campaign_handlers_bridge.go", Role: "campaign_list_patch_wiring", Domain: "campaign", Bridge: "campaign_handlers_bridge.go", Drained: true},
	{File: "campaign_wizard_bridge.go", Role: "campaign_wizard_publish_wiring", Domain: "campaign", Bridge: "campaign_wizard_bridge.go", Drained: true},
	{File: "campaign_import_bridge.go", Role: "import_wiring", Domain: "campaign,supply", Bridge: "campaign_import_bridge.go", Drained: true},
	{File: "campaign_worker_bridge.go", Role: "campaign_worker_wiring", Domain: "campaign", Bridge: "campaign_worker_bridge.go", Drained: true},
	{File: "dashboardadmin_bridge.go", Role: "dashboard_wiring", Domain: "dashboardadmin", Bridge: "dashboardadmin_bridge.go", Drained: true},
	{File: "flow_bridge.go", Role: "flow_crud_wiring", Domain: "flow", Bridge: "flow_bridge.go", Drained: true},

	{File: "governance_bridge.go", Role: "governance_wiring", Domain: "governance", Bridge: "governance_bridge.go", Drained: true},
	{File: "http_bridge.go", Role: "auth_session_wiring", Domain: "control/http,platformadmin", Bridge: "http_bridge.go", Drained: true},
	{File: "nodeadmin_bridge.go", Role: "node_scoring_wiring", Domain: "nodeadmin", Bridge: "nodeadmin_bridge.go", Drained: true},
	{File: "outbox_bridge.go", Role: "outbox_wiring", Domain: "outbox", Bridge: "outbox_bridge.go", Drained: true},
	{File: "platform_bridge.go", Role: "platform_config_wiring", Domain: "platformadmin", Bridge: "platform_bridge.go", Drained: true},
	{File: "platform_team_bridge.go", Role: "platform_team_telemetry_wiring", Domain: "platformadmin", Bridge: "platform_team_bridge.go", Drained: true},

	{File: "privacy_bridge.go", Role: "privacy_wiring", Domain: "privacy", Bridge: "privacy_bridge.go", Drained: true},
	{File: "reports_bridge.go", Role: "reports_catalog_wiring", Domain: "reports", Bridge: "reports_bridge.go", Drained: true},

	{File: "rtb_bridge.go", Role: "rtb_deals_floors_wiring", Domain: "rtbadmin", Bridge: "rtb_bridge.go", Drained: true},
	{File: "shard_bridge.go", Role: "shard_slot_wiring", Domain: "shardadmin", Bridge: "shard_bridge.go", Drained: true},

	{File: "supply_bridge.go", Role: "supply_wiring", Domain: "supply", Bridge: "supply_bridge.go", Drained: true},
}
