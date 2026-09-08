# UX scenario catalog (OpenAPI v0.6.0)

Source: `api/openapi/openapi.yaml` and `api/openapi/paths/*.yaml`.

Total atomic scenarios: **347** (one per HTTP operation).

Contract rules: `.cursor/rules/frontend-slop.mdc` (UX-A..N, EH-*).

## Summary by tag

| Tag | Count | Prefix |
|-----|------:|--------|
| audit | 2 | UX-AUD-* |
| auth | 4 | UX-ATH-* |
| automation | 6 | UX-ATN-* |
| billing | 23 | UX-BIL-* |
| brands | 8 | UX-BRD-* |
| campaigns | 51 | UX-CAM-* |
| command-palette | 5 | UX-CPL-* |
| consent | 1 | UX-CNS-* |
| cost-sync | 7 | UX-CSY-* |
| customers | 2 | UX-CST-* |
| dashboards | 7 | UX-DSH-* |
| disputes | 1 | UX-DSP-* |
| domains | 6 | UX-DOM-* |
| eula | 2 | UX-EUL-* |
| flows | 6 | UX-FLW-* |
| fraud-admin | 7 | UX-FRD-* |
| integration | 9 | UX-INT-* |
| landers | 12 | UX-LAN-* |
| license | 2 | UX-LIC-* |
| margin-guard | 4 | UX-MRG-* |
| meta | 3 | UX-MTA-* |
| ops | 29 | UX-OPS-* |
| platform-campaigns | 8 | UX-PLC-* |
| postbacks | 7 | UX-PST-* |
| public | 2 | UX-PBL-* |
| publisher | 2 | UX-PBR-* |
| recon | 1 | UX-RCN-* |
| report-schedules | 5 | UX-RSC-* |
| reports | 52 | UX-RPT-* |
| rtb | 10 | UX-RTB-* |
| selfserve | 8 | UX-SLF-* |
| settings | 4 | UX-SET-* |
| smart-alerts | 6 | UX-SMA-* |
| supply | 12 | UX-SPY-* |
| support | 2 | UX-SPT-* |
| team | 7 | UX-TEM-* |
| telegram | 13 | UX-TLG-* |
| traffic-optimizer | 6 | UX-TRO-* |
| views | 5 | UX-VIW-* |

## Scenario types

| Type | ID pattern | Meaning |
|------|------------|--------|
| Atomic | `UX-<TAG>-NNN` | Single OpenAPI operation = one user intent |
| Composite | `CMP-NNN` | Multi-step flow spanning several operations |
| Shell-only | route em dash | API-only, webhook, or no dedicated admin page yet |

## Composite flows (multi-operation)

| CMP | Flow | Tag | operationIds |
|-----|------|-----|-------------|
| CMP-001 | Bootstrap shell | meta | `sessionBootstrap, metaGet, sessionGet` |
| CMP-002 | Login session | auth | `authLogin, authMe, authRefresh, authLogout` |
| CMP-003 | Campaign onboarding wizard | campaigns | `campaignsOnboardingTemplates, campaignsWizardSessionGet, campaignsWizardSessionPost, campaignsForecast, campaignsPublish` |
| CMP-004 | Campaign list drill-down | campaigns | `campaignsList, campaignsListFacets, campaignsListMetrics, campaignsListMetricsTotals, campaignsGet` |
| CMP-005 | Campaign editor publish | campaigns | `campaignsEditorShell, campaignsPatch, campaignsValidatePatch, campaignsPublishCheck, campaignsPublish, campaignsSmoke` |
| CMP-006 | Campaign import/migrate | campaigns | `campaignsImportValidate, campaignsImportValidateJobCreate, campaignsImportValidateJobGet, campaignsImport, campaignsMigratePreview, campaignsMigrateImport` |
| CMP-007 | Hosted lander editor | landers | `landersGet, landersHostedEditorState, landersHostedUpload, landersHostedFileGet, landersHostedFilePut, landersHostedPublish, landersServePreview` |
| CMP-008 | Lander directory CRUD | landers | `landersList, landersCreate, landersUpdate, landersDelete` |
| CMP-009 | Flow builder | flows | `flowsList, flowsCreate, flowsGet, flowsUpdate, campaignsFlowValidate` |
| CMP-010 | Report browse + export | reports | `reportCatalog, reportCreateJob, reportGetJob, reportDownloadJob, reportCancelJob` |
| CMP-011 | Billing invoice lifecycle | billing | `billingListInvoices, billingGetInvoice, billingGetInvoicePdf, billingListInvoiceDeliveries, billingRetryInvoiceDelivery, billingVoidInvoice` |
| CMP-012 | Customer wallet top-up | selfserve | `selfserveCreatePaymentIntent, billingCustomerWallet, billingCustomerBalance` |
| CMP-013 | Team invite | team | `teamMembersList, teamInviteMember, publicInviteAccept` |
| CMP-014 | Domain onboarding | domains | `domainsAdd, domainsSslSetup, domainsProbe, domainsList` |
| CMP-015 | Postback setup | postbacks | `postbacksListConfig, postbacksUpdateConfig, postbacksTestConfig, postbacksSnapshot` |
| CMP-016 | Cost sync setup | cost-sync | `costSyncListNetworks, costSyncUpsertCredential, costSyncRunManual, costSyncListHistory` |
| CMP-017 | Command palette | command-palette | `commandPaletteSearch, commandPaletteListRoutes, commandPaletteRecordRecent, commandPaletteListRecents` |
| CMP-018 | Ops incident response | ops | `opsHome, opsIncidents, opsListDlqInbox, opsRetryDlqInbox, opsDoctor` |
| CMP-019 | Fraud label + override | fraud-admin | `listFraudPresets, listFraudLabels, upsertFraudLabel, createFraudOverride, getFraudDecision` |
| CMP-020 | RTB deal management | rtb | `rtbListDeals, rtbCreateDeal, rtbPatchDeal, rtbValidateBidRequest, rtbShadowDiff` |

## Mutation script matrix (admin UI)

| UX rule | Applies to |
|---------|------------|
| UX-A | One primary CTA per create intent (no duplicate Create paths) |
| UX-B | Modal dismiss without side effects until explicit Save |
| UX-C | Navigate to detail only after 2xx with entity id |
| UX-D | Toast/success only after apiConfirmed 2xx |
| UX-E | Delete: confirm dialog; handle 409 conflict with message |
| UX-F | Async job: poll GET .../jobs/{id} then download |
| UX-G | List refresh: coalesced refreshToken, ErrorBlock on failure |
| UX-H | RBAC: server 403, not client-only nav hide |
| UX-I | Dry-run endpoints preview before mutating PUT/POST |
| UX-J | Export/download: trigger only when job status=complete |
| UX-K | Wizard session: persist via POST session before commit |
| UX-L | Bulk mutate: show partial failure summary from response |
| UX-M | Hosted editor: save file PUT before publish POST |
| UX-N | Import validate before import commit |

## Atomic scenarios by domain

### audit (2)

| UX-ID | operationId | HTTP | Intent | Admin route |
|-------|-------------|------|--------|-------------|
| UX-AUD-001 | `auditList` | GET `/api/v1/audit` | List admin audit log entries | — |
| UX-AUD-002 | `auditExport` | GET `/api/v1/audit/export` | Export audit log as CSV | — |

### auth (4)

| UX-ID | operationId | HTTP | Intent | Admin route |
|-------|-------------|------|--------|-------------|
| UX-ATH-001 | `authLogin` | POST `/api/v1/auth/login` | Login with email and password | /login |
| UX-ATH-002 | `authLogout` | POST `/api/v1/auth/logout` | Revoke refresh token and clear session cookies | — |
| UX-ATH-003 | `authMe` | GET `/api/v1/auth/me` | Current authenticated user | — |
| UX-ATH-004 | `authRefresh` | POST `/api/v1/auth/refresh` | Rotate access token from refresh cookie | — |

### automation (6)

| UX-ID | operationId | HTTP | Intent | Admin route |
|-------|-------------|------|--------|-------------|
| UX-ATN-001 | `automationListPresets` | GET `/api/v1/automation/presets` | List bundled automation rule presets | — |
| UX-ATN-002 | `automationListRules` | GET `/api/v1/automation/rules` | List automation rules for a customer | — |
| UX-ATN-003 | `automationCreateRule` | POST `/api/v1/automation/rules` | Create automation rule | — |
| UX-ATN-004 | `automationDeleteRule` | DELETE `/api/v1/automation/rules/{id}` | Delete automation rule | — |
| UX-ATN-005 | `automationUpdateRule` | PUT `/api/v1/automation/rules/{id}` | Update automation rule | — |
| UX-ATN-006 | `automationDryRunRule` | POST `/api/v1/automation/rules/{id}/dry-run` | Dry-run automation rule against current ClickHouse metrics | — |

### billing (23)

| UX-ID | operationId | HTTP | Intent | Admin route |
|-------|-------------|------|--------|-------------|
| UX-BIL-001 | `billingCryptoWebhook` | POST `/api/v1/billing/crypto/webhook` | Crypto payment provider webhook | — |
| UX-BIL-002 | `billingCreateExport` | POST `/api/v1/billing/exports` | Enqueue async customer ledger export job | — |
| UX-BIL-003 | `billingGetExportJob` | GET `/api/v1/billing/exports/{job_id}` | Poll export job status | — |
| UX-BIL-004 | `billingDownloadExport` | GET `/api/v1/billing/exports/{job_id}/download` | Download completed export file | — |
| UX-BIL-005 | `billingInvariant` | GET `/api/v1/billing/invariant` | Ledger balance invariant check | — |
| UX-BIL-006 | `billingListInvoices` | GET `/api/v1/billing/invoices` | List invoices for customer or admin fleet | /billing |
| UX-BIL-007 | `billingPreviewInvoice` | POST `/api/v1/billing/invoices/preview` | Dry-run invoice generation for a billing month | — |
| UX-BIL-008 | `billingGetInvoice` | GET `/api/v1/billing/invoices/{id}` | Get invoice by id | — |
| UX-BIL-009 | `billingListInvoiceDeliveries` | GET `/api/v1/billing/invoices/{id}/deliveries` | List invoice email/webhook delivery attempts | — |
| UX-BIL-010 | `billingRetryInvoiceDelivery` | POST `/api/v1/billing/invoices/{id}/deliveries/retry` | Retry failed invoice delivery | — |
| UX-BIL-011 | `billingInvoiceLedgerLines` | GET `/api/v1/billing/invoices/{id}/ledger-lines` | Cursor-paged ledger lines backing an invoice | — |
| UX-BIL-012 | `billingGetInvoicePdf` | GET `/api/v1/billing/invoices/{id}/pdf` | Download invoice PDF | — |
| UX-BIL-013 | `billingVoidInvoice` | POST `/api/v1/billing/invoices/{id}/void` | Void a finalized invoice | — |
| UX-BIL-014 | `billingSummary` | GET `/api/v1/billing/summary` | Fleet billing KPI snapshot | — |
| UX-BIL-015 | `billingCustomerBalance` | GET `/api/v1/customers/{id}/balance` | Customer wallet balance with recent ledger slice | — |
| UX-BIL-016 | `billingCustomerBalanceExport` | GET `/api/v1/customers/{id}/balance/export` | Stream customer ledger CSV export | — |
| UX-BIL-017 | `billingCustomerForecast` | GET `/api/v1/customers/{id}/billing/forecast` | Month-to-date spend forecast for customer | — |
| UX-BIL-018 | `billingCustomerStatement` | GET `/api/v1/customers/{id}/billing/statement` | Customer billing statement for calendar month | — |
| UX-BIL-019 | `billingCustomerLedger` | GET `/api/v1/customers/{id}/ledger` | Paginated customer balance ledger | — |
| UX-BIL-020 | `billingCustomerPayments` | GET `/api/v1/customers/{id}/payments` | Payment intent history for customer | — |
| UX-BIL-021 | `billingGetTaxProfile` | GET `/api/v1/customers/{id}/tax-profile` | Get customer tax profile | — |
| UX-BIL-022 | `billingPutTaxProfile` | PUT `/api/v1/customers/{id}/tax-profile` | Replace customer tax profile | — |
| UX-BIL-023 | `billingCustomerWallet` | GET `/api/v1/customers/{id}/wallet` | Customer wallet and payment provider readiness | — |

### brands (8)

| UX-ID | operationId | HTTP | Intent | Admin route |
|-------|-------------|------|--------|-------------|
| UX-BRD-001 | `brandCreativesDelete` | DELETE `/api/v1/brand-creatives/{id}` | Delete brand creative | — |
| UX-BRD-002 | `brandCreativesPatch` | PATCH `/api/v1/brand-creatives/{id}` | Update brand creative | — |
| UX-BRD-003 | `brandsList` | GET `/api/v1/brands` | List brands for customer | /brands |
| UX-BRD-004 | `brandsCreate` | POST `/api/v1/brands` | Create brand | /brands |
| UX-BRD-005 | `getBrandsById` | GET `/api/v1/brands/{id}` | Route stub; schema not yet documented | — |
| UX-BRD-006 | `brandsGet` | GET `/api/v1/brands/{id}` | Get brand by id | — |
| UX-BRD-007 | `brandCreativesList` | GET `/api/v1/brands/{id}/creatives` | List brand creatives | — |
| UX-BRD-008 | `brandCreativesCreate` | POST `/api/v1/brands/{id}/creatives` | Create brand creative | — |

### campaigns (51)

| UX-ID | operationId | HTTP | Intent | Admin route |
|-------|-------------|------|--------|-------------|
| UX-CAM-001 | `campaignsList` | GET `/api/v1/campaigns` | List campaigns | /campaigns |
| UX-CAM-002 | `campaignsBulkMutate` | POST `/api/v1/campaigns/bulk` | Bulk pause, resume, or archive campaigns | — |
| UX-CAM-003 | `campaignsExportBatch` | GET `/api/v1/campaigns/export` | Export multiple campaign bundles in one request | — |
| UX-CAM-004 | `campaignsImport` | POST `/api/v1/campaigns/import` | Import campaign from export bundle | — |
| UX-CAM-005 | `campaignsImportValidate` | POST `/api/v1/campaigns/import/validate` | Synchronous migration validation without committing campaigns | — |
| UX-CAM-006 | `campaignsImportValidateJobCreate` | POST `/api/v1/campaigns/import/validate/jobs` | Enqueue async migration validation job | — |
| UX-CAM-007 | `campaignsImportValidateJobGet` | GET `/api/v1/campaigns/import/validate/jobs/{id}` | Read async migration validation job status | — |
| UX-CAM-008 | `getCampaignsListFacets` | GET `/api/v1/campaigns/list-facets` | Route stub; schema not yet documented | — |
| UX-CAM-009 | `campaignsListFacets` | GET `/api/v1/campaigns/list-facets` | Campaign list filter facets | — |
| UX-CAM-010 | `campaignsListMetrics` | GET `/api/v1/campaigns/metrics` | Batch campaign list metrics | — |
| UX-CAM-011 | `getCampaignsMetricsTotals` | GET `/api/v1/campaigns/metrics-totals` | Route stub; schema not yet documented | — |
| UX-CAM-012 | `campaignsListMetricsTotals` | GET `/api/v1/campaigns/metrics-totals` | Aggregated campaign list metrics for current filters | — |
| UX-CAM-013 | `campaignsMigrateImport` | POST `/api/v1/campaigns/migrate/import` | Import campaigns from external tracker migration payload | — |
| UX-CAM-014 | `campaignsMigratePreview` | POST `/api/v1/campaigns/migrate/preview` | Preview external tracker migration payload without importing | — |
| UX-CAM-015 | `campaignsMigratePullImport` | POST `/api/v1/campaigns/migrate/pull/import` | Pull migration payload from external tracker and import | — |
| UX-CAM-016 | `campaignsMigratePullPreview` | POST `/api/v1/campaigns/migrate/pull/preview` | Pull migration payload from external tracker and preview | — |
| UX-CAM-017 | `campaignsMigrateListSources` | GET `/api/v1/campaigns/migrate/sources` | List supported campaign migration source kinds | — |
| UX-CAM-018 | `campaignsOnboardingTemplates` | GET `/api/v1/campaigns/onboarding-templates` | List bundled campaign onboarding templates | — |
| UX-CAM-019 | `getCampaignsTargetCountries` | GET `/api/v1/campaigns/target-countries` | Route stub; schema not yet documented | — |
| UX-CAM-020 | `campaignsListTargetCountries` | GET `/api/v1/campaigns/target-countries` | List distinct campaign target countries | — |
| UX-CAM-021 | `campaignsWizardSessionGet` | GET `/api/v1/campaigns/wizard/session` | Read onboarding wizard session state | — |
| UX-CAM-022 | `campaignsWizardSessionPost` | POST `/api/v1/campaigns/wizard/session` | Create, update, or commit onboarding wizard session | — |
| UX-CAM-023 | `campaignsGet` | GET `/api/v1/campaigns/{id}` | Get campaign by id | — |
| UX-CAM-024 | `campaignsPatch` | PATCH `/api/v1/campaigns/{id}` | Partial campaign update | — |
| UX-CAM-025 | `campaignsClone` | POST `/api/v1/campaigns/{id}/clone` | Clone campaign with flow and postback config | — |
| UX-CAM-026 | `campaignsClonePreview` | POST `/api/v1/campaigns/{id}/clone-preview` | Preview campaign clone options | — |
| UX-CAM-027 | `campaignsListConversionMappings` | GET `/api/v1/campaigns/{id}/conversion-mappings` | List inbound status to payout mappings | — |
| UX-CAM-028 | `campaignsReplaceConversionMappings` | PUT `/api/v1/campaigns/{id}/conversion-mappings` | Replace all conversion mappings for campaign | — |
| UX-CAM-029 | `campaignsDiff` | GET `/api/v1/campaigns/{id}/diff` | Compare campaign against another campaign | — |
| UX-CAM-030 | `campaignsEditorShell` | GET `/api/v1/campaigns/{id}/editor` | Campaign editor section shell | /campaigns/:id/edit |
| UX-CAM-031 | `campaignsListEvents` | GET `/api/v1/campaigns/{id}/events` | List campaign conversion events | — |
| UX-CAM-032 | `campaignsExport` | GET `/api/v1/campaigns/{id}/export` | Export campaign bundle for import elsewhere | — |
| UX-CAM-033 | `campaignsFlowValidate` | POST `/api/v1/campaigns/{id}/flow/validate` | Validate campaign flow paths | — |
| UX-CAM-034 | `campaignsGetFraud` | GET `/api/v1/campaigns/{id}/fraud` | Get campaign fraud thresholds | — |
| UX-CAM-035 | `campaignsPatchFraud` | PATCH `/api/v1/campaigns/{id}/fraud` | Update campaign fraud thresholds | — |
| UX-CAM-036 | `campaignsFraudEditorSummary` | GET `/api/v1/campaigns/{id}/fraud-editor` | Campaign fraud editor cards for editor shell | — |
| UX-CAM-037 | `campaignsPreviewFraud` | POST `/api/v1/campaigns/{id}/fraud/preview` | Preview fraud threshold impact (7d shadow scores) | — |
| UX-CAM-038 | `campaignsGeoSummary` | GET `/api/v1/campaigns/{id}/geo-summary` | Campaign geo targeting summary for editor | — |
| UX-CAM-039 | `campaignsGetIntegrationHealth` | GET `/api/v1/campaigns/{id}/integration-health` | Integration readiness checklist for a campaign | — |
| UX-CAM-040 | `campaignsIntegrationPanel` | GET `/api/v1/campaigns/{id}/integration-panel` | Campaign integration health panel | — |
| UX-CAM-041 | `campaignsMacroPreview` | POST `/api/v1/campaigns/{id}/macro-preview` | Preview resolved click URL macros | — |
| UX-CAM-042 | `campaignsGetMargin` | GET `/api/v1/campaigns/{id}/margin` | Campaign margin guard window snapshot | — |
| UX-CAM-043 | `campaignsAssignOwner` | PUT `/api/v1/campaigns/{id}/owner` | Assign campaign owner user | — |
| UX-CAM-044 | `campaignsPlacementBlockSuggestions` | GET `/api/v1/campaigns/{id}/placement-block-suggestions` | Suggest placements to block based on IVT | — |
| UX-CAM-045 | `campaignsBlockPlacement` | POST `/api/v1/campaigns/{id}/placement-blocks` | Block a placement for a campaign | — |
| UX-CAM-046 | `campaignsPublish` | POST `/api/v1/campaigns/{id}/publish` | Validate and activate a paused campaign | — |
| UX-CAM-047 | `campaignsPublishCheck` | GET `/api/v1/campaigns/{id}/publish-check` | Evaluate publish gate without changing status | — |
| UX-CAM-048 | `campaignsSmoke` | POST `/api/v1/campaigns/{id}/smoke` | Synthetic click smoke test for campaign redirect chain | — |
| UX-CAM-049 | `campaignsGetStats` | GET `/api/v1/campaigns/{id}/stats` | Campaign stats time series | — |
| UX-CAM-050 | `campaignsValidatePatch` | POST `/api/v1/campaigns/{id}/validate` | Dry-run campaign patch validation | — |
| UX-CAM-051 | `campaignsForecast` | POST `/api/v1/forecast/campaign` | Campaign spend/impression forecast (wizard) | — |

### command-palette (5)

| UX-ID | operationId | HTTP | Intent | Admin route |
|-------|-------------|------|--------|-------------|
| UX-CPL-001 | `commandPaletteOpen` | POST `/api/v1/command-palette/open` | Record command palette open event for metrics | — |
| UX-CPL-002 | `commandPaletteListRecents` | GET `/api/v1/command-palette/recents` | Recent command palette selections for the authenticated user | — |
| UX-CPL-003 | `commandPaletteRecordRecent` | POST `/api/v1/command-palette/recents` | Record a command palette selection | — |
| UX-CPL-004 | `commandPaletteListRoutes` | GET `/api/v1/command-palette/routes` | Static admin navigation entries for command palette | — |
| UX-CPL-005 | `commandPaletteSearch` | GET `/api/v1/command-palette/search` | Global command palette typeahead search | Ctrl+K |

### consent (1)

| UX-ID | operationId | HTTP | Intent | Admin route |
|-------|-------------|------|--------|-------------|
| UX-CNS-001 | `consentRecord` | POST `/api/v1/consent` | Record signed user consent | — |

### cost-sync (7)

| UX-ID | operationId | HTTP | Intent | Admin route |
|-------|-------------|------|--------|-------------|
| UX-CSY-001 | `costSyncListCredentials` | GET `/api/v1/cost-sync/credentials` | List stored cost sync credentials | — |
| UX-CSY-002 | `costSyncDeleteCredential` | DELETE `/api/v1/cost-sync/credentials/{network}` | Delete stored credentials for a customer and network | — |
| UX-CSY-003 | `costSyncUpsertCredential` | PUT `/api/v1/cost-sync/credentials/{network}` | Create or update network credentials | — |
| UX-CSY-004 | `costSyncListHistory` | GET `/api/v1/cost-sync/history` | List cost sync run history | — |
| UX-CSY-005 | `costSyncListNetworks` | GET `/api/v1/cost-sync/networks` | List supported ad networks and credential field schemas | — |
| UX-CSY-006 | `costSyncRunManual` | POST `/api/v1/cost-sync/run` | Trigger a manual cost sync run | — |
| UX-CSY-007 | `costSyncSnapshot` | GET `/api/v1/cost-sync/snapshot` | Cost sync networks, credentials, and history snapshot | — |

### customers (2)

| UX-ID | operationId | HTTP | Intent | Admin route |
|-------|-------------|------|--------|-------------|
| UX-CST-001 | `customersList` | GET `/api/v1/customers` | List customers | — |
| UX-CST-002 | `customersGet` | GET `/api/v1/customers/{id}` | Get customer by id | — |

### dashboards (7)

| UX-ID | operationId | HTTP | Intent | Admin route |
|-------|-------------|------|--------|-------------|
| UX-DSH-001 | `dashboardAccountant` | GET `/api/v1/dashboards/accountant` | Accountant role dashboard | — |
| UX-DSH-002 | `dashboardAdops` | GET `/api/v1/dashboards/adops` | Ad ops role dashboard | — |
| UX-DSH-003 | `dashboardBuyer` | GET `/api/v1/dashboards/buyer` | Buyer role dashboard | /dashboards/buyer |
| UX-DSH-004 | `dashboardCampaign` | GET `/api/v1/dashboards/campaign/{id}` | Single-campaign dashboard | — |
| UX-DSH-005 | `dashboardCfo` | GET `/api/v1/dashboards/cfo` | CFO role dashboard | — |
| UX-DSH-006 | `dashboardFraud` | GET `/api/v1/dashboards/fraud` | Fraud analyst dashboard | — |
| UX-DSH-007 | `dashboardOperator` | GET `/api/v1/dashboards/operator` | Operator incident dashboard | — |

### disputes (1)

| UX-ID | operationId | HTTP | Intent | Admin route |
|-------|-------------|------|--------|-------------|
| UX-DSP-001 | `disputesList` | GET `/api/v1/disputes` | List payment disputes | — |

### domains (6)

| UX-ID | operationId | HTTP | Intent | Admin route |
|-------|-------------|------|--------|-------------|
| UX-DOM-001 | `domainsList` | GET `/api/v1/domains` | List tracked domain health rows | /domains |
| UX-DOM-002 | `domainsAdd` | POST `/api/v1/domains` | Register custom tracking domain | /domains |
| UX-DOM-003 | `domainsPark` | POST `/api/v1/domains/park` | Park domain via Cloudflare DNS | — |
| UX-DOM-004 | `domainsDelete` | DELETE `/api/v1/domains/{hostname}` | Remove custom domain | — |
| UX-DOM-005 | `domainsProbe` | POST `/api/v1/domains/{hostname}/probe` | Run immediate domain health probe | — |
| UX-DOM-006 | `domainsSslSetup` | POST `/api/v1/domains/{hostname}/ssl/setup` | Trigger TLS certificate setup for hostname | — |

### eula (2)

| UX-ID | operationId | HTTP | Intent | Admin route |
|-------|-------------|------|--------|-------------|
| UX-EUL-001 | `eulaStatus` | GET `/api/v1/eula` | Current EULA acceptance status | — |
| UX-EUL-002 | `eulaAccept` | POST `/api/v1/eula/accept` | Accept EULA version | — |

### flows (6)

| UX-ID | operationId | HTTP | Intent | Admin route |
|-------|-------------|------|--------|-------------|
| UX-FLW-001 | `flowsList` | GET `/api/v1/flows` | List flows | /flows |
| UX-FLW-002 | `flowsCreate` | POST `/api/v1/flows` | Create flow | /flows |
| UX-FLW-003 | `flowsGet` | GET `/api/v1/flows/{id}` | Get flow by id | /flows/:id |
| UX-FLW-004 | `flowsUpdate` | PUT `/api/v1/flows/{id}` | Replace flow paths | /flows/:id |
| UX-FLW-005 | `offersList` | GET `/api/v1/offers` | List offers | /offers |
| UX-FLW-006 | `offersCreate` | POST `/api/v1/offers` | Create offer | /offers |

### fraud-admin (7)

| UX-ID | operationId | HTTP | Intent | Admin route |
|-------|-------------|------|--------|-------------|
| UX-FRD-001 | `getFraudDecision` | GET `/api/v1/fraud/decisions` | Explain fraud tier decision for IP hash | — |
| UX-FRD-002 | `listFraudIntegrations` | GET `/api/v1/fraud/integrations` | List third-party fraud integration status | — |
| UX-FRD-003 | `listFraudLabels` | GET `/api/v1/fraud/labels` | List ML manual labels for customer | — |
| UX-FRD-004 | `upsertFraudLabel` | POST `/api/v1/fraud/labels` | Upsert ML manual label | — |
| UX-FRD-005 | `bulkUpsertFraudLabels` | POST `/api/v1/fraud/labels/bulk` | Bulk upsert ML manual labels | — |
| UX-FRD-006 | `createFraudOverride` | POST `/api/v1/fraud/overrides` | Create per-campaign fraud override | — |
| UX-FRD-007 | `listFraudPresets` | GET `/api/v1/fraud/presets` | List global fraud sensitivity presets | — |

### integration (9)

| UX-ID | operationId | HTTP | Intent | Admin route |
|-------|-------------|------|--------|-------------|
| UX-INT-001 | `integrationApplyCampaignTemplates` | POST `/api/v1/campaigns/{id}/apply-templates` | Apply bundled traffic/affiliate templates to a campaign | — |
| UX-INT-002 | `integrationAffiliateStatusPresets` | GET `/api/v1/integration/affiliate-status-presets` | Bundled affiliate status mapping presets | — |
| UX-INT-003 | `integrationListSchemas` | GET `/api/v1/integration/schemas` | List custom integration schemas | — |
| UX-INT-004 | `integrationCreateSchema` | POST `/api/v1/integration/schemas` | Create integration schema document | — |
| UX-INT-005 | `integrationGetSchema` | GET `/api/v1/integration/schemas/{id}` | Get integration schema by id | — |
| UX-INT-006 | `integrationApplySchema` | POST `/api/v1/integration/schemas/{id}/apply` | Apply schema to a campaign | — |
| UX-INT-007 | `integrationSnapshot` | GET `/api/v1/integration/snapshot` | Integration schemas and bundled templates snapshot | — |
| UX-INT-008 | `integrationListTemplates` | GET `/api/v1/integration/templates` | List bundled integration templates | — |
| UX-INT-009 | `integrationImportTemplates` | POST `/api/v1/integration/templates/import` | Import bundled templates into integration_schemas | — |

### landers (12)

| UX-ID | operationId | HTTP | Intent | Admin route |
|-------|-------------|------|--------|-------------|
| UX-LAN-001 | `landersList` | GET `/api/v1/landers` | List landers | /landers |
| UX-LAN-002 | `landersCreate` | POST `/api/v1/landers` | Create lander | /landers |
| UX-LAN-003 | `landersDelete` | DELETE `/api/v1/landers/{id}` | Delete lander | — |
| UX-LAN-004 | `landersGet` | GET `/api/v1/landers/{id}` | Get lander by id | — |
| UX-LAN-005 | `landersUpdate` | PATCH `/api/v1/landers/{id}` | Update lander metadata | — |
| UX-LAN-006 | `landersHostedEditorState` | GET `/api/v1/landers/{id}/hosted-editor` | Hosted lander editor file tree | /landers/:id/editor |
| UX-LAN-007 | `landersHostedFileGet` | GET `/api/v1/landers/{id}/hosted-files/{path...}` | Read hosted lander draft file | /landers/:id/editor |
| UX-LAN-008 | `landersHostedFilePut` | PUT `/api/v1/landers/{id}/hosted-files/{path...}` | Save hosted lander draft file | /landers/:id/editor |
| UX-LAN-009 | `landersHostedPublish` | POST `/api/v1/landers/{id}/hosted-publish` | Publish hosted lander draft | /landers/:id/editor |
| UX-LAN-010 | `landersHostedUpload` | POST `/api/v1/landers/{id}/hosted-upload` | Upload hosted lander ZIP | /landers/:id/editor |
| UX-LAN-011 | `landersServePreview` | GET `/lp-preview/{lander_id}/{path...}` | Serve draft hosted lander preview | — |
| UX-LAN-012 | `landersServePublished` | GET `/lp/{lander_id}/{path...}` | Serve published hosted lander static files | — |

### license (2)

| UX-ID | operationId | HTTP | Intent | Admin route |
|-------|-------------|------|--------|-------------|
| UX-LIC-001 | `licenseApply` | POST `/api/v1/license/apply` | Apply license JWT token | — |
| UX-LIC-002 | `licenseStatus` | GET `/api/v1/license/status` | JWT license diagnostics | — |

### margin-guard (4)

| UX-ID | operationId | HTTP | Intent | Admin route |
|-------|-------------|------|--------|-------------|
| UX-MRG-001 | `marginGuardListActivity` | GET `/api/v1/margin-guard/activity` | List margin guard activity for a campaign | — |
| UX-MRG-002 | `marginGuardRemoveOverride` | POST `/api/v1/margin-guard/overrides` | Clear placement override for a campaign | — |
| UX-MRG-003 | `marginGuardListPolicies` | GET `/api/v1/margin-guard/policies` | List margin guard policies for a campaign | — |
| UX-MRG-004 | `marginGuardCreatePolicy` | POST `/api/v1/margin-guard/policies` | Create margin guard policy | — |

### meta (3)

| UX-ID | operationId | HTTP | Intent | Admin route |
|-------|-------------|------|--------|-------------|
| UX-MTA-001 | `metaGet` | GET `/api/v1/meta` | Product metadata for admin shell bootstrap | — |
| UX-MTA-002 | `sessionGet` | GET `/api/v1/session` | Permission-filtered nav and session hints for admin shell | — |
| UX-MTA-003 | `sessionBootstrap` | GET `/api/v1/session/bootstrap` | Authenticated user, session shell, and EULA flags in one response | — |

### ops (29)

| UX-ID | operationId | HTTP | Intent | Admin route |
|-------|-------------|------|--------|-------------|
| UX-OPS-001 | `opsRemoveBlacklist` | DELETE `/api/v1/ops/blacklist` | Remove IP from fraud blacklist | — |
| UX-OPS-002 | `opsListBlacklist` | GET `/api/v1/ops/blacklist` | List fraud blacklist entries | — |
| UX-OPS-003 | `opsAddBlacklist` | POST `/api/v1/ops/blacklist` | Block IP on fraud blacklist | — |
| UX-OPS-004 | `opsConsentProofs` | GET `/api/v1/ops/consent/proofs` | GDPR consent proof audit log | — |
| UX-OPS-005 | `opsDashboardMetrics` | GET `/api/v1/ops/dashboard/metrics` | Prometheus metric time series for ops dashboard | — |
| UX-OPS-006 | `opsDashboardStream` | GET `/api/v1/ops/dashboard/stream` | SSE stream of ops dashboard events | — |
| UX-OPS-007 | `opsDashboardSummary` | GET `/api/v1/ops/dashboard/summary` | Ops dashboard KPI summary | — |
| UX-OPS-008 | `opsListDlq` | GET `/api/v1/ops/dlq` | Fan-out DLQ entries across shards | — |
| UX-OPS-009 | `opsListDlqInbox` | GET `/api/v1/ops/dlq/inbox` | Unified DLQ inbox across sources | — |
| UX-OPS-010 | `opsRetryDlqInbox` | POST `/api/v1/ops/dlq/inbox/{id}/retry` | Retry DLQ inbox entry | — |
| UX-OPS-011 | `opsRetryDlq` | POST `/api/v1/ops/dlq/{id}/retry` | Retry DLQ entry by id | — |
| UX-OPS-012 | `opsDoctor` | GET `/api/v1/ops/doctor` | Platform health doctor checks | — |
| UX-OPS-013 | `opsDomainRotation` | GET `/api/v1/ops/domains/rotation` | Tracking domain rotation state | — |
| UX-OPS-014 | `opsTlsAllowedList` | GET `/api/v1/ops/domains/tls-allowed` | Hostnames allowed for TLS setup | — |
| UX-OPS-015 | `opsTlsAllowedHost` | GET `/api/v1/ops/domains/{hostname}/tls-allowed` | Check whether hostname is TLS-allowed | — |
| UX-OPS-016 | `opsPatchFraudPreset` | PATCH `/api/v1/ops/fraud/presets/{name}` | Update global fraud preset thresholds | — |
| UX-OPS-017 | `opsStackHealthSnapshot` | GET `/api/v1/ops/health/snapshot` | Cold-path stack health snapshot | — |
| UX-OPS-018 | `opsHome` | GET `/api/v1/ops/home` | Ops home doctor, stack health, and dashboard summary | — |
| UX-OPS-019 | `opsIncidents` | GET `/api/v1/ops/incidents` | Shard and outbox incident snapshot | — |
| UX-OPS-020 | `opsMlModelStatus` | GET `/api/v1/ops/ml-model` | ML model deployment status | — |
| UX-OPS-021 | `opsMlModelEval` | GET `/api/v1/ops/ml-model/eval` | ML model offline eval metrics | — |
| UX-OPS-022 | `opsListMlLabels` | GET `/api/v1/ops/ml-model/labels` | Fleet ML manual labels (operator scope) | — |
| UX-OPS-023 | `opsAddMlLabel` | POST `/api/v1/ops/ml-model/labels` | Add fleet-scoped ML manual label | — |
| UX-OPS-024 | `opsListOutbox` | GET `/api/v1/ops/outbox` | Paginated outbox event tail | — |
| UX-OPS-025 | `opsRolesReload` | POST `/api/v1/ops/roles/reload` | Reload RBAC role map from Postgres | — |
| UX-OPS-026 | `opsRum` | GET `/api/v1/ops/rum` | Real user monitoring samples | — |
| UX-OPS-027 | `opsRumIngest` | POST `/api/v1/ops/rum` | Ingest RUM beacon batch | — |
| UX-OPS-028 | `opsListShards` | GET `/api/v1/ops/shards` | Redis shard health matrix | — |
| UX-OPS-029 | `opsShard0Catchup` | POST `/api/v1/ops/shards/0/catchup` | Run shard-0 config catch-up worker | — |

### platform-campaigns (8)

| UX-ID | operationId | HTTP | Intent | Admin route |
|-------|-------------|------|--------|-------------|
| UX-PLC-001 | `platformCampaignsListLinks` | GET `/api/v1/platform-campaigns/links` | List external ad platform campaign links | — |
| UX-PLC-002 | `platformCampaignsDeleteLink` | DELETE `/api/v1/platform-campaigns/links/{campaign_id}/{network}` | Delete platform campaign link | — |
| UX-PLC-003 | `platformCampaignsUpsertLink` | PUT `/api/v1/platform-campaigns/links/{campaign_id}/{network}` | Upsert platform campaign link | — |
| UX-PLC-004 | `platformCampaignsRefreshLink` | POST `/api/v1/platform-campaigns/links/{campaign_id}/{network}/refresh` | Refresh external status for a platform link | — |
| UX-PLC-005 | `platformCampaignsSyncRun` | POST `/api/v1/platform-campaigns/sync-run` | Trigger manual platform status sync for a campaign | — |
| UX-PLC-006 | `platformCampaignsSetBudget` | POST `/api/v1/platform-campaigns/{campaign_id}/budget` | Set external daily budget cap | — |
| UX-PLC-007 | `platformCampaignsPause` | POST `/api/v1/platform-campaigns/{campaign_id}/pause` | Pause external ad platform campaign | — |
| UX-PLC-008 | `platformCampaignsResume` | POST `/api/v1/platform-campaigns/{campaign_id}/resume` | Resume external ad platform campaign | — |

### postbacks (7)

| UX-ID | operationId | HTTP | Intent | Admin route |
|-------|-------------|------|--------|-------------|
| UX-PST-001 | `postbacksListCampaignStatus` | GET `/api/v1/postbacks/campaign-status` | List per-campaign postback health | — |
| UX-PST-002 | `postbacksListConfig` | GET `/api/v1/postbacks/config` | List outbound postback configs | — |
| UX-PST-003 | `postbacksUpdateConfig` | PUT `/api/v1/postbacks/config/{campaign_id}` | Upsert campaign postback config | — |
| UX-PST-004 | `postbacksTestConfig` | POST `/api/v1/postbacks/config/{campaign_id}/test` | Dry-run postback dispatch for a campaign | — |
| UX-PST-005 | `postbacksListDlq` | GET `/api/v1/postbacks/dlq` | List postback dead-letter queue entries | — |
| UX-PST-006 | `postbacksRetryDlq` | POST `/api/v1/postbacks/dlq/{id}/retry` | Re-enqueue a DLQ postback | — |
| UX-PST-007 | `postbacksSnapshot` | GET `/api/v1/postbacks/snapshot` | Postbacks configs, DLQ, and campaign status snapshot | — |

### public (2)

| UX-ID | operationId | HTTP | Intent | Admin route |
|-------|-------------|------|--------|-------------|
| UX-PBL-001 | `publicActivate` | POST `/api/v1/public/activate` | Activate owner account with license JWT | /activate |
| UX-PBL-002 | `publicInviteAccept` | POST `/api/v1/public/invite/accept` | Accept team invite and set password | — |

### publisher (2)

| UX-ID | operationId | HTTP | Intent | Admin route |
|-------|-------------|------|--------|-------------|
| UX-PBR-001 | `publisherDashboard` | GET `/api/v1/publisher/dashboard` | Scoped publisher KPI dashboard | — |
| UX-PBR-002 | `publisherStatements` | GET `/api/v1/publisher/statements` | Publisher revenue statements | — |

### recon (1)

| UX-ID | operationId | HTTP | Intent | Admin route |
|-------|-------------|------|--------|-------------|
| UX-RCN-001 | `reconListRuns` | GET `/api/v1/recon/runs` | List reconciliation batch runs | — |

### report-schedules (5)

| UX-ID | operationId | HTTP | Intent | Admin route |
|-------|-------------|------|--------|-------------|
| UX-RSC-001 | `reportSchedulesList` | GET `/api/v1/report-schedules` | List report delivery schedules | — |
| UX-RSC-002 | `reportSchedulesCreate` | POST `/api/v1/report-schedules` | Create report schedule | — |
| UX-RSC-003 | `reportSchedulesDelete` | DELETE `/api/v1/report-schedules/{id}` | Delete report schedule | — |
| UX-RSC-004 | `reportSchedulesGet` | GET `/api/v1/report-schedules/{id}` | Get report schedule | — |
| UX-RSC-005 | `reportSchedulesUpdate` | PUT `/api/v1/report-schedules/{id}` | Update report schedule | — |

### reports (52)

| UX-ID | operationId | HTTP | Intent | Admin route |
|-------|-------------|------|--------|-------------|
| UX-RPT-001 | `reportCampaignGeoDevice` | GET `/api/v1/reports/campaign-geo-device` | Geo and device breakdown | — |
| UX-RPT-002 | `reportCampaignOverview` | GET `/api/v1/reports/campaign-overview` | Campaign overview rollup | — |
| UX-RPT-003 | `reportCampaignToggleCohort` | GET `/api/v1/reports/campaign-toggle-cohort` | Before and after metrics for campaign fraud toggle changes | — |
| UX-RPT-004 | `reportCatalog` | GET `/api/v1/reports/catalog` | Report catalog for current auth snapshot | /reports |
| UX-RPT-005 | `reportClickLog` | GET `/api/v1/reports/click-log` | Click log timeline | — |
| UX-RPT-006 | `reportClicks` | GET `/api/v1/reports/clicks` | Click log browse alias | — |
| UX-RPT-007 | `reportConversionTypePayout` | GET `/api/v1/reports/conversion-type-payout` | Conversion type payout rollup | — |
| UX-RPT-008 | `reportCostSyncCoverage` | GET `/api/v1/reports/cost-sync-coverage` | Cost sync attribution coverage | — |
| UX-RPT-009 | `reportCustomerFraudByDimension` | GET `/api/v1/reports/customer-fraud-by-dimension` | Customer fraud concentration by dimension | — |
| UX-RPT-010 | `reportCustomerFraudByType` | GET `/api/v1/reports/customer-fraud-by-type` | Customer fraud categories aggregated by campaign | — |
| UX-RPT-011 | `reportCustomerFraudEvidence` | GET `/api/v1/reports/customer-fraud-evidence` | Signed redacted fraud evidence for customer disputes | — |
| UX-RPT-012 | `reportCustomerPortfolio` | GET `/api/v1/reports/customer-portfolio` | Customer portfolio KPIs | — |
| UX-RPT-013 | `reportDataQuality` | GET `/api/v1/reports/data-quality` | Ingest data quality signals | — |
| UX-RPT-014 | `reportDaypartHeatmap` | GET `/api/v1/reports/daypart-heatmap` | Hour-of-day performance heatmap | — |
| UX-RPT-015 | `reportDiscrepancyBuySell` | GET `/api/v1/reports/discrepancy-buy-sell` | Buy vs sell discrepancy | — |
| UX-RPT-016 | `reportEdgeParity` | GET `/api/v1/reports/edge-parity` | Edge vs tracker parity drift | — |
| UX-RPT-017 | `reportFilterRejects` | GET `/api/v1/reports/filter-rejects` | Unified filter reject counts | — |
| UX-RPT-018 | `reportFraudBreakdown` | GET `/api/v1/reports/fraud-breakdown` | Fraud tier breakdown | — |
| UX-RPT-019 | `reportFraudEvidencePack` | GET `/api/v1/reports/fraud-evidence-pack` | Signed fraud evidence pack for CPA disputes | — |
| UX-RPT-020 | `reportGeoRoi` | GET `/api/v1/reports/geo-roi` | ROI by country | — |
| UX-RPT-021 | `reportIvtBySource` | GET `/api/v1/reports/ivt-by-source` | IVT rate by traffic source | — |
| UX-RPT-022 | `reportCreateJob` | POST `/api/v1/reports/jobs` | Enqueue async report export job | — |
| UX-RPT-023 | `reportCancelJob` | DELETE `/api/v1/reports/jobs/{id}` | Cancel pending report export job | — |
| UX-RPT-024 | `reportGetJob` | GET `/api/v1/reports/jobs/{id}` | Poll report export job status | — |
| UX-RPT-025 | `reportDownloadJob` | GET `/api/v1/reports/jobs/{id}/download` | Download completed report export | — |
| UX-RPT-026 | `reportKeywords` | GET `/api/v1/reports/keywords` | Keyword performance | — |
| UX-RPT-027 | `reportLayerDesyncDrilldown` | GET `/api/v1/reports/layer-desync-drilldown` | Layer desync fraud reason drilldown with hourly series | — |
| UX-RPT-028 | `reportLayerDesyncSummary` | GET `/api/v1/reports/layer-desync-summary` | Cross-layer desync fraud counts | — |
| UX-RPT-029 | `reportMlFeatureSpikes` | GET `/api/v1/reports/ml/feature-spikes` | ML feature spike detection | — |
| UX-RPT-030 | `reportMlScoreDistribution` | GET `/api/v1/reports/ml/score-distribution` | ML score distribution | — |
| UX-RPT-031 | `reportMlShadowDelta` | GET `/api/v1/reports/ml/shadow-delta` | Shadow vs live ML delta | — |
| UX-RPT-032 | `reportPacingDrift` | GET `/api/v1/reports/pacing-drift` | Budget pacing drift vs plan | — |
| UX-RPT-033 | `reportPlacements` | GET `/api/v1/reports/placements` | Placement performance by zone | — |
| UX-RPT-034 | `reportPostbackReconciliation` | GET `/api/v1/reports/postback-reconciliation` | Postback vs ledger reconciliation | — |
| UX-RPT-035 | `reportRtbGeoDevice` | GET `/api/v1/reports/rtb/geo-device` | RTB geo and device stats | — |
| UX-RPT-036 | `reportRtbNoBidReasons` | GET `/api/v1/reports/rtb/no-bid-reasons` | RTB no-bid reason breakdown | — |
| UX-RPT-037 | `reportRtbOverview` | GET `/api/v1/reports/rtb/overview` | RTB auction overview | — |
| UX-RPT-038 | `reportRTTSplitTunnel` | GET `/api/v1/reports/rtt-split-tunnel` | RTT split-tunnel distribution | — |
| UX-RPT-039 | `reportSignalEffectiveness` | GET `/api/v1/reports/signal-effectiveness` | Wire signal block and silent-reject rates | — |
| UX-RPT-040 | `reportSilentRejectFunnel` | GET `/api/v1/reports/silent-reject-impression-funnel` | Non-blocking fraud response funnel | — |
| UX-RPT-041 | `reportSourceQuality` | GET `/api/v1/reports/source-quality` | Sub-source quality scores | — |
| UX-RPT-042 | `reportSpendVelocity` | GET `/api/v1/reports/spend-velocity` | Spend velocity time series | — |
| UX-RPT-043 | `reportTelegram` | GET `/api/v1/reports/telegram` | Telegram Mini App rollup | — |
| UX-RPT-044 | `reportTelegramBots` | GET `/api/v1/reports/telegram/bots` | Telegram bot performance | — |
| UX-RPT-045 | `reportTelegramExport` | POST `/api/v1/reports/telegram/export` | Export Telegram report bundle | — |
| UX-RPT-046 | `reportTelegramFraud` | GET `/api/v1/reports/telegram/fraud` | Telegram fraud signals | — |
| UX-RPT-047 | `reportTelegramFunnel` | GET `/api/v1/reports/telegram/funnel` | Telegram conversion funnel | — |
| UX-RPT-048 | `reportTelegramPremium` | GET `/api/v1/reports/telegram/premium` | Telegram premium users | — |
| UX-RPT-049 | `reportTelegramSummary` | GET `/api/v1/reports/telegram/summary` | Telegram summary KPIs | — |
| UX-RPT-050 | `reportTrafficSources` | GET `/api/v1/reports/traffic-sources` | Traffic channel rollup | — |
| UX-RPT-051 | `reportTrueRoi` | GET `/api/v1/reports/true-roi` | True ROI after cost sync | — |
| UX-RPT-052 | `reportWireSignalBreakdown` | GET `/api/v1/reports/wire-signal-breakdown` | Wire signal fraud breakdown | — |

### rtb (10)

| UX-ID | operationId | HTTP | Intent | Admin route |
|-------|-------------|------|--------|-------------|
| UX-RTB-001 | `rtbListDeals` | GET `/api/v1/rtb/deals` | List RTB deals | — |
| UX-RTB-002 | `rtbCreateDeal` | POST `/api/v1/rtb/deals` | Create RTB deal | — |
| UX-RTB-003 | `rtbDeleteDeal` | DELETE `/api/v1/rtb/deals/{id}` | Delete RTB deal | — |
| UX-RTB-004 | `rtbGetDeal` | GET `/api/v1/rtb/deals/{id}` | Get RTB deal by id | — |
| UX-RTB-005 | `rtbPatchDeal` | PATCH `/api/v1/rtb/deals/{id}` | Update RTB deal | — |
| UX-RTB-006 | `rtbFloorsApply` | POST `/api/v1/rtb/floors/apply` | Apply floor optimizer suggestions | — |
| UX-RTB-007 | `rtbIntegrationProfile` | GET `/api/v1/rtb/integration-profile` | OpenRTB integration profile and endpoint hints | — |
| UX-RTB-008 | `rtbReconcileExport` | GET `/api/v1/rtb/reconcile/export` | RTB reconcile export with live gate | — |
| UX-RTB-009 | `rtbShadowDiff` | GET `/api/v1/rtb/shadow-diff` | Shadow vs live auction parity snapshot | — |
| UX-RTB-010 | `rtbValidateBidRequest` | POST `/api/v1/rtb/validate-bid-request` | Validate raw OpenRTB bid request JSON | — |

### selfserve (8)

| UX-ID | operationId | HTTP | Intent | Admin route |
|-------|-------------|------|--------|-------------|
| UX-SLF-001 | `selfserveCreateApiKey` | POST `/api/v1/selfserve/api-keys` | Mint self-serve API key (session required) | — |
| UX-SLF-002 | `selfserveBillingStatement` | GET `/api/v1/selfserve/billing/statement` | Self-serve billing statement for current customer | — |
| UX-SLF-003 | `selfserveCreateCampaign` | POST `/api/v1/selfserve/campaigns` | Create campaign from template | — |
| UX-SLF-004 | `selfservePauseCampaign` | POST `/api/v1/selfserve/campaigns/{id}/pause` | Pause campaign via self-serve API | — |
| UX-SLF-005 | `selfserveResumeCampaign` | POST `/api/v1/selfserve/campaigns/{id}/resume` | Resume campaign via self-serve API | — |
| UX-SLF-006 | `selfserveListInvoices` | GET `/api/v1/selfserve/invoices` | List invoices for authenticated self-serve customer | — |
| UX-SLF-007 | `selfserveCreatePaymentIntent` | POST `/api/v1/selfserve/payment-intents` | Create top-up payment intent | — |
| UX-SLF-008 | `selfserveListTemplates` | GET `/api/v1/selfserve/templates` | List campaign templates for self-serve create | — |

### settings (4)

| UX-ID | operationId | HTTP | Intent | Admin route |
|-------|-------------|------|--------|-------------|
| UX-SET-001 | `settingsPlatformGet` | GET `/api/v1/settings/platform` | Read platform install configuration | /settings |
| UX-SET-002 | `settingsPlatformPatch` | PATCH `/api/v1/settings/platform` | Merge platform configuration patch | /settings |
| UX-SET-003 | `settingsPlatformApply` | POST `/api/v1/settings/platform/apply` | Write platform config to install root on disk | — |
| UX-SET-004 | `settingsPlatformBootstrap` | POST `/api/v1/settings/platform/bootstrap` | First-run platform bootstrap | — |

### smart-alerts (6)

| UX-ID | operationId | HTTP | Intent | Admin route |
|-------|-------------|------|--------|-------------|
| UX-SMA-001 | `smartAlertsAckEvent` | POST `/api/v1/smart-alerts/events/{id}/ack` | Acknowledge a smart alert event | — |
| UX-SMA-002 | `smartAlertsListHistory` | GET `/api/v1/smart-alerts/history` | List smart alert firing history | — |
| UX-SMA-003 | `smartAlertsListRules` | GET `/api/v1/smart-alerts/rules` | List smart alert rules for a customer | — |
| UX-SMA-004 | `smartAlertsCreateRule` | POST `/api/v1/smart-alerts/rules` | Create smart alert rule | — |
| UX-SMA-005 | `smartAlertsDeleteRule` | DELETE `/api/v1/smart-alerts/rules/{id}` | Delete smart alert rule | — |
| UX-SMA-006 | `smartAlertsUpdateRule` | PATCH `/api/v1/smart-alerts/rules/{id}` | Update smart alert rule | — |

### supply (12)

| UX-ID | operationId | HTTP | Intent | Admin route |
|-------|-------------|------|--------|-------------|
| UX-SPY-001 | `supplyListAdsTxt` | GET `/api/v1/supply/ads-txt` | List ads.txt rows | — |
| UX-SPY-002 | `supplyCreateAdsTxt` | POST `/api/v1/supply/ads-txt` | Create ads.txt row | — |
| UX-SPY-003 | `supplyDeleteAdsTxt` | DELETE `/api/v1/supply/ads-txt/{id}` | Delete ads.txt row | — |
| UX-SPY-004 | `supplyUpdateAdsTxt` | PUT `/api/v1/supply/ads-txt/{id}` | Update ads.txt row | — |
| UX-SPY-005 | `supplyGetExportPath` | GET `/api/v1/supply/export-path` | Return nginx export path for supply files | — |
| UX-SPY-006 | `supplyPreviewAdsTxt` | GET `/api/v1/supply/preview/ads.txt` | Preview rendered ads.txt | — |
| UX-SPY-007 | `supplyPreviewSellersJSON` | GET `/api/v1/supply/preview/sellers.json` | Preview rendered sellers.json | — |
| UX-SPY-008 | `supplyListSellers` | GET `/api/v1/supply/sellers` | List sellers.json rows | — |
| UX-SPY-009 | `supplyCreateSeller` | POST `/api/v1/supply/sellers` | Create sellers.json row | — |
| UX-SPY-010 | `supplyDeleteSeller` | DELETE `/api/v1/supply/sellers/{id}` | Delete sellers.json row | — |
| UX-SPY-011 | `supplyUpdateSeller` | PUT `/api/v1/supply/sellers/{id}` | Update sellers.json row | — |
| UX-SPY-012 | `supplyGetValidation` | GET `/api/v1/supply/validation` | Validate sellers.json and ads.txt exports | — |

### support (2)

| UX-ID | operationId | HTTP | Intent | Admin route |
|-------|-------------|------|--------|-------------|
| UX-SPT-001 | `supportFeedbackCreate` | POST `/api/v1/support/feedback` | Submit operator feedback | — |
| UX-SPT-002 | `supportFeedbackMeta` | GET `/api/v1/support/feedback/meta` | Support feedback form metadata | — |

### team (7)

| UX-ID | operationId | HTTP | Intent | Admin route |
|-------|-------------|------|--------|-------------|
| UX-TEM-001 | `teamBudgetApprovalsList` | GET `/api/v1/team/budget-approvals` | Pending budget approval requests | — |
| UX-TEM-002 | `teamBudgetApprovalApprove` | POST `/api/v1/team/budget-approvals/{id}/approve` | Approve budget increase request | — |
| UX-TEM-003 | `teamBudgetApprovalDeny` | POST `/api/v1/team/budget-approvals/{id}/deny` | Deny budget increase request | — |
| UX-TEM-004 | `teamMembersList` | GET `/api/v1/team/members` | Paginated team members for a customer | — |
| UX-TEM-005 | `teamInviteMember` | POST `/api/v1/team/members` | Invite team member | — |
| UX-TEM-006 | `teamUpdateMember` | PATCH `/api/v1/team/members/{id}` | Update team member role or caps | — |
| UX-TEM-007 | `teamOverview` | GET `/api/v1/team/overview` | Team license and balance snapshot | — |

### telegram (13)

| UX-ID | operationId | HTTP | Intent | Admin route |
|-------|-------------|------|--------|-------------|
| UX-TLG-001 | `telegramListBots` | GET `/api/v1/telegram/bots` | List configured Telegram bots | — |
| UX-TLG-002 | `telegramGetBot` | GET `/api/v1/telegram/bots/{id}` | Get bot config for campaign | — |
| UX-TLG-003 | `telegramConfigureBot` | PUT `/api/v1/telegram/bots/{id}` | Upsert Telegram bot config | — |
| UX-TLG-004 | `telegramMintClick` | POST `/api/v1/telegram/clicks` | Mint click_id for Mini App session | — |
| UX-TLG-005 | `telegramCreateDeeplink` | POST `/api/v1/telegram/deeplink-tokens` | Create deeplink attribution token | — |
| UX-TLG-006 | `telegramGetDeeplink` | GET `/api/v1/telegram/deeplink-tokens/{token}` | Resolve deeplink token | — |
| UX-TLG-007 | `telegramListPostbacks` | GET `/api/v1/telegram/postbacks` | List Telegram postback URLs for campaign | — |
| UX-TLG-008 | `telegramCreatePostback` | POST `/api/v1/telegram/postbacks` | Create Telegram postback URL | — |
| UX-TLG-009 | `telegramDeletePostback` | DELETE `/api/v1/telegram/postbacks/{id}` | Delete Telegram postback | — |
| UX-TLG-010 | `telegramUpdatePostback` | PUT `/api/v1/telegram/postbacks/{id}` | Update Telegram postback URL | — |
| UX-TLG-011 | `telegramTestPostback` | POST `/api/v1/telegram/postbacks/{id}/test` | Dry-run Telegram postback dispatch | — |
| UX-TLG-012 | `telegramValidateInitData` | POST `/api/v1/telegram/validate` | Validate Telegram Mini App initData | — |
| UX-TLG-013 | `telegramWebhook` | POST `/api/v1/telegram/webhook/{bot_id}` | Telegram bot webhook receiver | — |

### traffic-optimizer (6)

| UX-ID | operationId | HTTP | Intent | Admin route |
|-------|-------------|------|--------|-------------|
| UX-TRO-001 | `trafficOptimizerListPresets` | GET `/api/v1/traffic-optimizer/presets` | List bundled traffic optimizer presets | — |
| UX-TRO-002 | `trafficOptimizerListRules` | GET `/api/v1/traffic-optimizer/rules` | List traffic optimizer rules for a customer | — |
| UX-TRO-003 | `trafficOptimizerCreateRule` | POST `/api/v1/traffic-optimizer/rules` | Create traffic optimizer rule | — |
| UX-TRO-004 | `trafficOptimizerDeleteRule` | DELETE `/api/v1/traffic-optimizer/rules/{id}` | Delete traffic optimizer rule | — |
| UX-TRO-005 | `trafficOptimizerUpdateRule` | PUT `/api/v1/traffic-optimizer/rules/{id}` | Update traffic optimizer rule | — |
| UX-TRO-006 | `trafficOptimizerDryRunRule` | POST `/api/v1/traffic-optimizer/rules/{id}/dry-run` | Dry-run traffic optimizer rule against current ClickHouse metrics | — |

### views (5)

| UX-ID | operationId | HTTP | Intent | Admin route |
|-------|-------------|------|--------|-------------|
| UX-VIW-001 | `listSavedViews` | GET `/api/v1/views` | List saved report views for customer | — |
| UX-VIW-002 | `createSavedView` | POST `/api/v1/views` | Create saved report view | — |
| UX-VIW-003 | `deleteSavedView` | DELETE `/api/v1/views/{id}` | Delete saved report view | — |
| UX-VIW-004 | `getSavedView` | GET `/api/v1/views/{id}` | Get saved report view | — |
| UX-VIW-005 | `updateSavedView` | PUT `/api/v1/views/{id}` | Update saved report view | — |
