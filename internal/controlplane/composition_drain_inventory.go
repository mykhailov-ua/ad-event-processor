package controlplane

type compositionDrainRow struct {
	File    string
	Role    string
	Domain  string
	Bridge  string
	Drained bool
}

var compositionDrainInventory = []compositionDrainRow{
	{File: "adminapi_wire.go", Role: "route_registry_wire", Domain: "shell", Drained: true},
	{File: "register.go", Role: "route_catalog", Domain: "shell", Drained: true},
	{File: "service.go", Role: "deps_lazy_inits", Domain: "shell", Drained: true},
	{File: "workers.go", Role: "background_workers", Domain: "shell", Drained: true},
	{File: "serve.go", Role: "http_server", Domain: "shell", Drained: true},
	{File: "handler.go", Role: "top_level_http", Domain: "shell", Drained: true},
	{File: "middleware.go", Role: "auth_middleware", Domain: "shell", Drained: true},
	{File: "errors.go", Role: "service_errors", Domain: "shell", Drained: true},
	{File: "license.go", Role: "license_watcher", Domain: "licensingadmin", Bridge: "licensing_bridge.go", Drained: true},
	{File: "admin_static.go", Role: "spa_bootstrap", Domain: "shell", Drained: true},
	{File: "test_helpers.go", Role: "test_constructors", Domain: "shell", Drained: true},
	{File: "doc.go", Role: "package_boundary", Domain: "shell", Drained: true},
	{File: "fraudadmin_bridge.go", Role: "fraud_wiring", Domain: "fraudadmin", Bridge: "fraudadmin_bridge.go", Drained: true},
	{File: "fraudadmin_hosts_bridge.go", Role: "fraud_hosts_wiring", Domain: "fraudadmin", Bridge: "fraudadmin_hosts_bridge.go", Drained: true},
	{File: "fraudadmin_service_bridge.go", Role: "fraud_service_wiring", Domain: "fraudadmin", Bridge: "fraudadmin_service_bridge.go", Drained: true},
	{File: "opsadmin_bridge.go", Role: "ops_wiring", Domain: "opsadmin", Bridge: "opsadmin_bridge.go", Drained: true},
	{File: "admin_misc_bridges.go", Role: "brand_margin_stub_wiring", Domain: "brand,marginguard", Bridge: "admin_misc_bridges.go", Drained: true},
	{File: "billing_bridge.go", Role: "billing_crypto_wiring", Domain: "billingadmin,payment", Bridge: "billing_bridge.go", Drained: true},
	{File: "campaign_handlers_bridge.go", Role: "campaign_crud_wiring", Domain: "campaign", Bridge: "campaign_handlers_bridge.go", Drained: true},
	{File: "campaign_supply_wizard_bridge.go", Role: "campaign_supply_wizard_wiring", Domain: "campaign,supply", Bridge: "campaign_supply_wizard_bridge.go", Drained: true},
	{File: "campaign_worker_bridge.go", Role: "campaign_worker_wiring", Domain: "campaign", Bridge: "campaign_worker_bridge.go", Drained: true},
	{File: "campaign_delivery_bridge.go", Role: "delivery_pacing_wiring", Domain: "campaign", Bridge: "campaign_delivery_bridge.go", Drained: true},
	{File: "campaign_autoscale_bridge.go", Role: "delivery_autoscale_wiring", Domain: "campaign", Bridge: "campaign_autoscale_bridge.go", Drained: true},
	{File: "campaign_bandit_bridge.go", Role: "delivery_bandit_wiring", Domain: "campaign,flow", Bridge: "campaign_bandit_bridge.go", Drained: true},
	{File: "campaign_import_bridge.go", Role: "import_export_wiring", Domain: "campaign", Bridge: "campaign_import_bridge.go", Drained: true},
	{File: "dashboardadmin_bridge.go", Role: "dashboard_wiring", Domain: "dashboardadmin", Bridge: "dashboardadmin_bridge.go", Drained: true},
	{File: "flow_bridge.go", Role: "flow_wiring", Domain: "flow", Bridge: "flow_bridge.go", Drained: true},
	{File: "governance_bridge.go", Role: "governance_wiring", Domain: "governance", Bridge: "governance_bridge.go", Drained: true},
	{File: "http_bridge.go", Role: "auth_session_wiring", Domain: "control/http,platformadmin", Bridge: "http_bridge.go", Drained: true},
	{File: "licensing_bridge.go", Role: "license_eula_wiring", Domain: "licensingadmin", Bridge: "licensing_bridge.go", Drained: true},
	{File: "nodeadmin_bridge.go", Role: "node_scoring_wiring", Domain: "nodeadmin", Bridge: "nodeadmin_bridge.go", Drained: true},
	{File: "outbox_bridge.go", Role: "outbox_wiring", Domain: "outbox", Bridge: "outbox_bridge.go", Drained: true},
	{File: "platform_bridge.go", Role: "platform_team_wiring", Domain: "platformadmin", Bridge: "platform_bridge.go", Drained: true},
	{File: "privacy_bridge.go", Role: "privacy_wiring", Domain: "privacy", Bridge: "privacy_bridge.go", Drained: true},
	{File: "reports_bridge.go", Role: "reports_job_wiring", Domain: "reports,reportjob", Bridge: "reports_bridge.go", Drained: true},
	{File: "rtb_bridge.go", Role: "rtb_wiring", Domain: "rtbadmin", Bridge: "rtb_bridge.go", Drained: true},
	{File: "settings_bridge.go", Role: "settings_wiring", Domain: "settingsadmin", Bridge: "settings_bridge.go", Drained: true},
	{File: "shard_bridge.go", Role: "shard_slot_wiring", Domain: "shardadmin", Bridge: "shard_bridge.go", Drained: true},
	{File: "shard_wires.go", Role: "shard_failover_wiring", Domain: "shardadmin", Bridge: "shard_bridge.go", Drained: true},
	{File: "supply_bridge.go", Role: "supply_wiring", Domain: "supply", Bridge: "supply_bridge.go", Drained: true},
}
