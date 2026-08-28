"""Rich metadata for admin UI milestone specs (deploy/vendor/ADMIN_*_MILESTONE*.md)."""

from __future__ import annotations

from dataclasses import dataclass, field


@dataclass
class Region:
    id: str
    component: str
    purpose: str
    source: str


@dataclass
class UrlParam:
    name: str
    api: str
    default: str
    notes: str = ""


@dataclass
class MilestoneSpec:
    filename: str
    slug: str
    title: str
    depends: str
    blocks: str
    pattern: str
    summary: str = ""
    route: str = ""
    apis: str = ""
    operation_id: str = ""
    schema_ref: str = ""
    handler: str = ""
    domain: str = ""
    legacy: str = ""
    nav_group: str = ""
    icon: str = ""
    permission: str = ""
    live: str = "true"
    in_scope: list[str] = field(default_factory=list)
    out_scope: list[str] = field(default_factory=list)
    not_on_page: list[str] = field(default_factory=list)
    api_gaps: list[tuple[str, str, str]] = field(default_factory=list)
    stop_triggers: list[str] = field(default_factory=list)
    contract_rows: list[tuple[str, str, str]] = field(default_factory=list)
    regions: list[Region] = field(default_factory=list)
    grid_cols: str = ""
    grid_ascii: str = ""
    grid_invariants: list[tuple[str, str]] = field(default_factory=list)
    url_params: list[UrlParam] = field(default_factory=list)
    sortable_cols: list[str] = field(default_factory=list)
    nonsortable_cols: list[str] = field(default_factory=list)
    fetch_example: str = ""
    files: list[tuple[str, str]] = field(default_factory=list)
    legacy_remove: list[str] = field(default_factory=list)
    risks: list[tuple[str, str, str]] = field(default_factory=list)
    slop: list[tuple[str, str, str]] = field(default_factory=list)
    laziness: list[tuple[str, str, str]] = field(default_factory=list)
    forbidden: list[str] = field(default_factory=list)
    pitfalls: list[tuple[str, str]] = field(default_factory=list)
    impl_steps: list[tuple[str, str, str, str]] = field(default_factory=list)
    sla_rows: list[tuple[str, str, str, str]] = field(default_factory=list)
    manual_checks: list[tuple[str, str, str]] = field(default_factory=list)
    extra_verify: list[str] = field(default_factory=list)
    gap_route: bool = False
    tabs: list[tuple[str, str, str]] = field(default_factory=list)  # tab_id, label, api


FOUNDATION_DEPS = (
    "`admin_contract_gate`, `admin_tokens`, `admin_shell`, `admin_page_chrome`"
)
PAGE_CHROME_DEPS = "`admin_contract_gate`, `admin_tokens`, `admin_shell`, `admin_page_chrome`"


def _dir_invariants(cols_var: str) -> list[tuple[str, str]]:
    return [
        ("Page grid", "`grid-template-rows: auto auto auto 1fr auto`; `min-height: 0` on content"),
        ("Row subgrid", f"Each row uses `{cols_var}`"),
        ("Max width", "`max-width: var(--page-max-width)` on page root"),
        ("Scroll", "Content region scrolls; chrome/toolbar/filter/footer in page grid"),
    ]


def _default_dir_files(domain: str, page: str) -> list[tuple[str, str]]:
    return [
        (f"web/src/pages/{page}", "Compose: useSearchParams + fetch + sections"),
        (f"web/src/ui/{domain}/{domain}_directory.tsx", "Section assembly"),
        (f"web/src/ui/{domain}/{domain}_toolbar.tsx", "Toolbar region"),
        (f"web/src/ui/{domain}/{domain}_filter.tsx", "FilterPanel draft + Apply"),
        (f"web/src/ui/{domain}/{domain}_grid.tsx", "role=grid + rows"),
        (f"web/src/ui/{domain}/{domain}_directory.module.css", "Page section grid"),
        (f"web/src/ui/{domain}/{domain}_grid.module.css", "Column template"),
        (f"web/src/helpers/{domain}_api.ts", "list(params, signal)"),
        ("web/src/types/generated/openapi.d.ts", "Generated types"),
    ]


def _default_dir_impl(domain: str, handler: str) -> list[tuple[str, str, str, str]]:
    return [
        ("1", handler or "OpenAPI + handler", "Confirm list op query params match URL", "Manual or test: param change affects response"),
        ("2", "make openapi-types", "Regenerate openapi.d.ts", "npm run typecheck passes"),
        ("3", f"web/src/helpers/{domain}_api.ts", "Typed list fetch", "Compiles; no hand-written DTO"),
        ("4", f"web/src/ui/{domain}/*", "Sections per 4.1–4.4", "check_ui_surface_gate.sh pass"),
        ("5", f"web/src/pages/*", "URL sync compose", "No table_sort / client filter on items[]"),
        ("6", "web/src/app_routes.tsx", "Route loads", "Lazy import resolves"),
    ]


def _default_sla() -> list[tuple[str, str, str, str]]:
    return [
        ("List fetch", "Cold admin", "N/A hot-path SLA", "Not /track"),
        ("Render", "Initial paint", "Skeleton in grid only", "No layout shift on chrome"),
        ("Scroll", "50 rows", "< 16 ms/frame", "Profiler optional"),
    ]


def _default_manual(domain: str, page: str) -> list[tuple[str, str, str]]:
    return [
        ("No client sort", f"rg 'table_sort|sortRows' web/src/pages/{page} web/src/ui/{domain}/", "no matches"),
        ("Pagination", "Manual: change offset in URL", "refetch; total stable"),
        ("Error", "Manual: block API", "ErrorBlock visible"),
    ]


DIRECTORIES: list[MilestoneSpec] = [
    MilestoneSpec(
        filename="ADMIN_DIRECTORY_MILESTONE_CAMPAIGNS.md",
        slug="admin_directory_campaigns",
        title="ADMIN_DIRECTORY_MILESTONE_CAMPAIGNS",
        depends=PAGE_CHROME_DEPS,
        blocks="`admin_detail_campaign`",
        pattern="admin_directory_pattern",
        summary="Reference directory #2 after customers. Bulk actions + multi-filter list.",
        route="/campaigns",
        apis="GET /api/v1/campaigns",
        operation_id="campaignsList",
        schema_ref="CampaignListResponse (`api/openapi/components/schemas/campaign.yaml`)",
        handler="internal/campaign/handlers.go — campaignsList",
        domain="campaigns",
        legacy="campaigns_page.tsx",
        nav_group="Commercial → Campaigns",
        icon="megaphone",
        permission="campaigns:read or campaigns:read:masked",
        in_scope=[
            "Rebuild `/campaigns` under `web/src/ui/campaigns/`",
            "URL-driven filters: customer_id, status, q, sort, order, pacing_mode (per OpenAPI)",
            "Grid: name (link), status_label + status_tone chip, budget_display or budget_limit, pacing summary, updated_at",
            "Row link → `/campaigns/{id}`; touchCustomerContext when customer scoped",
            "Bulk selection + POST `/api/v1/campaigns/bulk` when permitted",
            "Buyer bound customer: default customer_id filter from session",
        ],
        out_scope=[
            "Campaign detail (`ADMIN_DETAIL_MILESTONE_CAMPAIGN.md`)",
            "Wizard / migrate routes",
            "Client-side buyer dashboard merge (legacy fetches buyer dashboard for health — move to server labels or separate widget milestone)",
            "RecentCustomers widget (defer)",
        ],
        not_on_page=["RecentCustomers", "client sortRows on fetched page"],
        api_gaps=[
            (
                "status_label / status_tone",
                "Handler sets json status_label; not in OpenAPI Campaign schema",
                "Use status + displayLabel in UI until OpenAPI extended; no invented tone enum",
            ),
            (
                "Buyer health badges",
                "Legacy parallelAll fetchBuyerDashboard for CampaignHealthBadge sort",
                "Remove buyer dashboard fetch; list Campaign fields only",
            ),
            (
                "budget_display",
                "Campaign schema has budget_limit string only",
                "formatUsdDecimal(budget_limit) in UI — no budget_display field",
            ),
            (
                "freshness_label",
                "Not on CampaignListResponse",
                "Omit FreshnessBadge until envelope extended",
            ),
            (
                "Bulk delete",
                "campaignsBulkMutate enum is pause|resume only",
                "No delete in bulk bar; per-row delete out of scope",
            ),
        ],
        stop_triggers=[
            "Wrapper stack (`CampaignsDirectoryChrome`) — revert",
            "Bulk action without OpenAPI body schema — stop and fix contract first",
        ],
        contract_rows=[
            (
                "List op",
                "GET /api/v1/campaigns — campaignsList",
                "customer_id, status, q, sort (name|updated_at|spend), order, pacing_mode, limit, offset",
            ),
            (
                "Bulk op",
                "POST /api/v1/campaigns/bulk — campaignsBulkMutate",
                "body: action pause|resume, campaign_ids[]",
            ),
            ("Row", "Campaign schema — campaign.yaml", "id, name, status, budget_limit, current_spend, pacing_mode, updated_at"),
            ("List envelope", "CampaignListResponse", "items, total, limit, offset, filters_applied, sort"),
            ("RBAC", "x-permissions", "campaigns:read, campaigns:read:masked"),
            ("Handler", "internal/campaign/handlers.go", "campaignsList"),
        ],
        regions=[
            Region("chrome", "PageChrome", 'Title "Campaigns"; subtitle total', "list.total"),
            Region("toolbar", "CampaignsToolbar", "Create wizard link, migrate link, import — permission gated", "static routes + permissions"),
            Region("filters", "CampaignsFilterPanel", "customer_id, status, q, pacing_mode, sort, order; Apply → URL", "URL params"),
            Region("bulk", "BulkActionBar", "Pause/resume/delete selection → bulk API", "POST .../bulk"),
            Region("content", "CampaignsGrid", "role=grid", "items[]"),
            Region("col_name", "grid cell", "Name + link to detail", "item.name, item.id"),
            Region("col_status", "StatusBadge", "status_label + status_tone", "API only"),
            Region("col_budget", "grid cell", "budget_display or formatted budget_limit", "API string/micros"),
            Region("col_pacing", "grid cell", "Pacing summary from API", "handler field"),
            Region("col_updated", "grid cell", "updated_at locale", "item.updated_at"),
            Region("footer", "PaginationBar", "limit/offset/total", "envelope"),
            Region("error", "ErrorBlock", "List failure", "fetch error"),
        ],
        grid_cols="minmax(14rem,2fr) 9rem 9rem 7rem 9rem",
        grid_ascii="""```
┌ PageChrome ─────────────────────────────────────────────┐
│ Campaigns                         {total} total         │
├ Toolbar ────────────────────────────────────────────────┤
│ [Wizard] [Migrate] [Import]                             │
├ FilterPanel ─────────────────────────────────────────────┤
│ Customer [▼] Status [▼] Search [____] Sort [▼] [Apply] │
├ Bulk (when selection) ──────────────────────────────────┤
│ [Pause] [Resume] …                                      │
├ Content (role=grid) ──────────────────────────────────────┤
│ --campaigns-cols: minmax(14rem,2fr) 9rem 9rem 7rem 9rem │
│ Name | Status | Budget | Pacing | Updated               │
├ Footer ───────────────────────────────────────────────────┤
│ PaginationBar                                           │
└─────────────────────────────────────────────────────────┘
```""",
        grid_invariants=_dir_invariants("--campaigns-cols")
        + [
            ("Sortable headers", "name, updated_at, spend (OpenAPI enum) only"),
            ("Checkbox column", "Leading column for bulk; does not affect --campaigns-cols data columns"),
        ],
        url_params=[
            UrlParam("limit", "limit", "50"),
            UrlParam("offset", "offset", "0"),
            UrlParam("customer_id", "customer_id", "", "from URL; buyer session default"),
            UrlParam("status", "status", "", "empty = all"),
            UrlParam("q", "q", "", "server search"),
            UrlParam("sort", "sort", "updated_at", "name | updated_at | spend"),
            UrlParam("order", "order", "desc", "asc | desc"),
            UrlParam("pacing_mode", "pacing_mode", "", "optional filter"),
        ],
        sortable_cols=["name", "updated_at", "spend"],
        nonsortable_cols=["status", "budget", "pacing"],
        fetch_example="GET /api/v1/campaigns?limit=50&offset=0&customer_id={uuid}&status=active&q=test&sort=name&order=asc",
        files=_default_dir_files("campaigns", "campaigns_page.tsx")
        + [
            ("web/src/ui/campaigns/campaigns_bulk_bar.tsx", "Bulk selection + campaignsBulkMutate"),
            ("web/src/ui/campaigns/campaigns_bulk_bar.module.css", "Bulk bar grid"),
            ("web/src/helpers/campaign_actions.ts", "pause/resume single row if needed"),
            ("web/src/helpers/campaign_admin_api.ts", "import/clone — toolbar only when permitted"),
        ],
        legacy_remove=["../components/*", "../lib/table_sort.js", "../types/campaign.js", "buyer dashboard merge for sort"],
        risks=[
            ("Client sortRows", "Legacy campaigns_page.tsx", "rg table_sort on route"),
            ("Buyer health merge", "parallelAll buyer dashboard", "List API labels only"),
            ("Bulk without schema", "Invented action names", "OpenAPI POST .../bulk body"),
            ("Masked read leak", "Show full customer on masked role", "RBAC + masked fields from API"),
        ],
        slop=[
            ("Import in toolbar without API", "Dead button", "Hide until OpenAPI POST import"),
            ("Pause per-row only", "N+1 pause calls", "Bulk endpoint when multi-select"),
        ],
        laziness=[
            ("Keep buildUrl without q/sort", "Minimal URL sync", "All OpenAPI query params in URL"),
        ],
        forbidden=[
            '"Server search" if q removed from OpenAPI',
            '"Bulk wired" without POST .../bulk in same PR',
            "sortRows on items[] for any column",
        ],
        pitfalls=[
            ("Client filter on items[]", "Forbidden; URL refetch only"),
            ("Buyer dashboard side fetch", "Health badges from parallel GET dashboards/buyer — remove"),
            ("sortRows on items[]", "Legacy table_sort — URL sort/order refetch"),
            ("Bulk delete invented", "OpenAPI bulk is pause|resume only — no delete action"),
            ("Bulk without confirm", "Confirm modal before campaignsBulkMutate POST"),
            ("Double status chrome", "One StatusBadge per cell; no nested frames"),
            ("Search UI without q", "q exists in OpenAPI — wire URL param or omit field"),
            ("status_label without OpenAPI", "Use status string + displayLabel until schema extended"),
            ("Wrapper stack", "No CampaignsDirectoryChrome — flat ui/campaigns/ sections"),
            ("Portal filter listbox", "Inline drop on sort/status selects"),
            ("N+1 pause per row", "Use POST /api/v1/campaigns/bulk for multi-select"),
            ("Import without idempotency", "POST import requires Idempotency-Key header"),
            ("Buyer scope leak", "Default customer_id in URL for bound buyer session"),
            ("RecentCustomers on page", "Defer widget — not directory pattern"),
        ],
        impl_steps=_default_dir_impl("campaigns", "api/openapi/paths/campaigns.yaml")
        + [
            (
                "7",
                "POST /api/v1/campaigns/bulk",
                "BulkActionBar → campaignsBulkMutate body (pause|resume)",
                "2xx + list refetch",
            ),
            (
                "8",
                "Legacy cleanup",
                "Remove table_sort, buyer dashboard merge, components/ imports",
                "rg table_sort web/src/pages/campaigns_page.tsx empty",
            ),
        ],
        sla_rows=_default_sla(),
        manual_checks=_default_manual("campaigns", "campaigns_page.tsx")
        + [
            ("Bulk pause", "Manual: select 2 rows → Pause", "campaignsBulkMutate 2xx; list refetch"),
            ("Buyer scope", "Manual: buyer session opens /campaigns", "customer_id defaulted in URL"),
            ("Search", "Manual: q=foo in URL", "server filters rows; no client filter"),
            ("Sort", "Manual: sort=spend&order=desc", "row order changes"),
            ("Masked read", "Manual: campaigns:read:masked session", "no PII leak in grid cells"),
        ],
        extra_verify=["rg 'sortRows' web/src/pages/campaigns_page.tsx"],
    ),
    MilestoneSpec(
        filename="ADMIN_DIRECTORY_MILESTONE_FLOWS.md",
        slug="admin_directory_flows",
        title="ADMIN_DIRECTORY_MILESTONE_FLOWS",
        depends=PAGE_CHROME_DEPS,
        blocks="`admin_detail_flow_builder`",
        pattern="admin_directory_pattern (hub tabs)",
        summary="Campaign flows hub at /campaigns/flows — landers, offers, flows tabs with create rows; flow row links to builder.",
        route="/campaigns/flows",
        apis="GET /api/v1/landers; GET /api/v1/offers; GET /api/v1/flows; POST landersCreate, offersCreate, flowsCreate",
        operation_id="flowsList",
        schema_ref="Lander[], Offer[], Flow[] arrays — campaign.yaml",
        handler="internal/flow/handlers.go — landersList, offersList, flowsList",
        domain="flows",
        legacy="campaign_flows_page.tsx",
        nav_group="Commercial → Campaigns → Flows",
        icon="git-branch",
        permission="campaigns:read",
        in_scope=[
            "Rebuild `/campaigns/flows` under `web/src/ui/flows/`",
            "Tab hub: landers | offers | flows (URL `?tab=` sync)",
            "List tables per tab from GET landers/offers/flows (full array — no pagination)",
            "Inline create forms per tab (POST) when campaigns:write",
            "Flow row → `/campaigns/flows/{id}/builder`",
            "Lander ZIP upload → POST landers/{id}/hosted-upload when on landers tab",
        ],
        out_scope=[
            "Flow builder editor (`ADMIN_DETAIL_MILESTONE_FLOW_BUILDER.md`)",
            "Hosted editor file tree (`ADMIN_DETAIL_MILESTONE_LANDER.md`)",
            "Client-side path validation (server validate on save)",
            "Server pagination (APIs return full arrays today)",
        ],
        not_on_page=["Client pagination", "sortRows on arrays", "Monolithic campaign_flows_page copy"],
        api_gaps=[
            (
                "No ListEnvelope",
                "landersList, offersList, flowsList return bare arrays",
                "No PaginationBar; document full-fetch; window if N>500 (react.mdc G9)",
            ),
            (
                "No list query params",
                "No limit/offset/sort/q on flow/lander/offer list ops",
                "No filter panel until OpenAPI adds params",
            ),
            (
                "Flow paths shape",
                "Flow.paths is array or string in schema",
                "Use summarizeFlowPaths helper; do not invent path DSL",
            ),
            (
                "Tab hub vs pure directory",
                "Legacy is 3-tab hub not single grid",
                "Milestone ships tab hub; do not collapse to flows-only grid without operator ask",
            ),
        ],
        stop_triggers=[
            "Wrapper stack (`FlowsHubChrome`) — revert",
            "Invented list filters without OpenAPI query params",
        ],
        contract_rows=[
            ("Landers list", "GET /api/v1/landers — landersList", "array of Lander"),
            ("Offers list", "GET /api/v1/offers — offersList", "array of Offer"),
            ("Flows list", "GET /api/v1/flows — flowsList", "array of Flow"),
            ("Create lander", "POST /api/v1/landers — landersCreate", "CreateLanderRequest"),
            ("Create offer", "POST /api/v1/offers — offersCreate", "CreateOfferRequest"),
            ("Create flow", "POST /api/v1/flows — flowsCreate", "CreateFlowRequest"),
            ("Hosted upload", "POST /api/v1/landers/{id}/hosted-upload", "ZIP upload on landers tab"),
            ("RBAC", "x-permissions", "campaigns:read / campaigns:write"),
        ],
        regions=[
            Region("chrome", "PageChrome", 'Title "Campaign flows"', "static"),
            Region("tabs", "FlowsTabBar", "landers | offers | flows", "URL ?tab="),
            Region("toolbar_landers", "LandersToolbar", "Create lander + ZIP upload", "POST landers + hosted-upload"),
            Region("toolbar_offers", "OffersToolbar", "Create offer", "POST offers"),
            Region("toolbar_flows", "FlowsToolbar", "Create flow", "POST flows"),
            Region("content_landers", "LandersGrid", "name, url, id, created_at", "GET landers[]"),
            Region("content_offers", "OffersGrid", "name, url, id", "GET offers[]"),
            Region("content_flows", "FlowsGrid", "name, paths summary, created_at, builder link", "GET flows[]"),
            Region("col_builder", "grid cell", "Open builder → /campaigns/flows/{id}/builder", "item.id"),
            Region("create_forms", "InlineCreateRow", "Per-tab POST forms", "OpenAPI create bodies"),
            Region("error", "ErrorBlock", "Any tab fetch/create failure", "fetch error"),
            Region("loading", "tab skeleton", "placeholder rows per active tab", "loading state"),
            Region("empty", "EmptyState", "per-tab empty copy", "array length 0"),
        ],
        grid_cols="minmax(12rem,2fr) 10rem 8rem 8rem",
        grid_ascii="""```
┌ PageChrome ─────────────────────────────────────────────┐
│ Campaign flows                                          │
├ TabBar ─────────────────────────────────────────────────┤
│ [Landers] [Offers] [Flows]                              │
├ Toolbar (per tab) ──────────────────────────────────────┤
│ [Create …] [Upload ZIP] (landers tab only)              │
├ Content (role=grid) ────────────────────────────────────┤
│ --flows-cols: minmax(12rem,2fr) 10rem 8rem 8rem         │
│ Name | URL/Paths | Created | Actions                    │
├ (no footer — full array fetch) ─────────────────────────┤
└─────────────────────────────────────────────────────────┘
```""",
        grid_invariants=_dir_invariants("--flows-cols")
        + [
            ("No pagination footer", "APIs return full arrays — no fake PaginationBar"),
            ("Tab in URL", "?tab=landers|offers|flows"),
            ("No client sort", "No sort headers until OpenAPI sort param"),
        ],
        url_params=[
            UrlParam("tab", "tab", "landers", "landers | offers | flows — UI only until API tab scope"),
        ],
        sortable_cols=[],
        nonsortable_cols=["name", "url", "paths", "created_at"],
        fetch_example="GET /api/v1/flows (parallel: GET /api/v1/landers, GET /api/v1/offers)",
        files=[
            ("web/src/pages/campaign_flows_page.tsx", "Compose: tab URL + parallel fetch"),
            ("web/src/ui/flows/flows_hub.tsx", "Tab hub section assembly"),
            ("web/src/ui/flows/flows_hub.module.css", "Hub section grid"),
            ("web/src/ui/flows/landers_panel.tsx", "Landers grid + create + upload"),
            ("web/src/ui/flows/offers_panel.tsx", "Offers grid + create"),
            ("web/src/ui/flows/flows_panel.tsx", "Flows grid + create + builder link"),
            ("web/src/ui/flows/*.module.css", "Per-panel column templates"),
            ("web/src/helpers/flows_api.ts", "fetchLanders/Offers/Flows + create helpers"),
            ("web/src/types/generated/openapi.d.ts", "Generated types"),
        ],
        legacy_remove=[
            "../components/tab_bar.js",
            "../components/breadcrumbs.js on page",
            "Monolithic inline forms in campaign_flows_page.tsx",
        ],
        risks=[
            ("Client sort on arrays", "Legacy none but agents add sortRows", "No sort until OpenAPI param"),
            ("Fake pagination", "PaginationBar without total", "Omit footer — document api_gap"),
            ("Partial error swallow", "Promise.all one tab fails silently", "ErrorBlock on any tab failure"),
            ("Create toast before 2xx", "Optimistic create", "apiConfirmed after POST 201"),
            ("Path validation in browser", "parseFlowPaths business rules", "Server flows validate endpoint on builder"),
        ],
        slop=[
            ("Flows-only grid", "Drops landers/offers tabs", "Preserve 3-tab hub per legacy route"),
            ("Copy campaign_flows_page", "700-line monolith", "ui/flows/* sections"),
            ("Portal tab listbox", "Flex tab bar", "TabBar grid section"),
            ("ZIP upload without lander id", "Dead upload button", "Select lander row first"),
        ],
        laziness=[
            ("Keep components/ TabBar", "Skip ui/flows", "Domain folder per frontend-modular.mdc"),
            ("Single fetch flows only", "Ignore landers/offers", "Parallel fetch all three tabs"),
            ("Skip URL tab param", "useState tab only", "?tab= sync on refresh"),
        ],
        forbidden=[
            '"Directory paginated" when APIs return full arrays',
            '"Server sort" without OpenAPI sort param on list ops',
            "Create form fields not on Create*Request schemas",
            "Builder link using wrong id field",
        ],
        pitfalls=[
            ("Fake pagination", "No limit/offset in OpenAPI — no PaginationBar"),
            ("Client sort on landers[]", "Forbidden — no sort param"),
            ("Tab state not in URL", "Refresh loses tab — use ?tab="),
            ("Monolith copy", "Reuse 500-line campaign_flows_page"),
            ("Toast before create 2xx", "apiConfirmed pattern"),
            ("ZIP upload wrong lander", "Require selected lander id"),
            ("Flow paths editor on hub", "Belongs in builder milestone"),
            ("Parallel fetch partial fail", "ErrorBlock not empty tab"),
            ("Wrapper FlowsHubChrome", "Flat ui/flows/ sections"),
            ("Flex tab layout", "CSS Grid sections only"),
            ("Invented offer URL rules", "Only http(s) check if mirrored from legacy — server validates"),
            ("Builder link typo", "Use item.id UUID not array index"),
        ],
        impl_steps=[
            ("1", "api/openapi/paths/campaigns.yaml", "Confirm landers/offers/flows ops + create bodies", "openapi_gate"),
            ("2", "make openapi-types", "Regenerate openapi.d.ts", "typecheck"),
            ("3", "web/src/helpers/flows_api.ts", "Typed list + create helpers", "no hand DTO"),
            ("4", "web/src/ui/flows/*", "Tab panels per 4.1", "check_ui_surface_gate.sh"),
            ("5", "web/src/pages/campaign_flows_page.tsx", "URL ?tab= + compose", "no components/ imports"),
            ("6", "web/src/app_routes.tsx", "Confirm /campaigns/flows route", "lazy import resolves"),
            ("7", "Builder link", "Row → /campaigns/flows/{id}/builder", "manual navigation"),
            ("8", "Legacy cleanup", "Remove inline tables from old page", "rg TabBar web/src/pages/campaign_flows_page.tsx"),
        ],
        sla_rows=_default_sla()
        + [
            ("Full array fetch", "Admin cold path", "Window rows if N>500", "react.mdc G9 threshold"),
        ],
        manual_checks=_default_manual("flows", "campaign_flows_page.tsx")
        + [
            ("Tab URL", "Manual: ?tab=flows refresh", "stays on flows tab"),
            ("Create lander", "Manual: POST create", "201 + list refetch"),
            ("Builder link", "Manual: open flow row", "navigates to builder route"),
            ("Error", "Manual: block API", "ErrorBlock visible"),
            ("No pagination", "rg PaginationBar web/src/ui/flows/", "no matches"),
        ],
        extra_verify=["rg 'table_sort' web/src/pages/campaign_flows_page.tsx web/src/ui/flows/"],
    ),
    MilestoneSpec(
        filename="ADMIN_DIRECTORY_MILESTONE_AUDIT.md",
        slug="admin_directory_audit",
        title="ADMIN_DIRECTORY_MILESTONE_AUDIT",
        depends=PAGE_CHROME_DEPS,
        blocks="—",
        pattern="admin_directory_pattern",
        summary="Audit log directory with CSV export; array body + X-Total-Count header pagination.",
        route="/audit",
        apis="GET /api/v1/audit — auditList; GET /api/v1/audit/export — auditExport",
        operation_id="auditList",
        schema_ref="AuditLog[] + X-Total-Count header — platform.yaml",
        handler="internal/platformadmin/audit_logs.go — auditList",
        domain="audit",
        legacy="audit_page.tsx",
        nav_group="Settings → Audit",
        icon="list",
        permission="audit:read",
        in_scope=[
            "Rebuild `/audit` under `web/src/ui/audit/`",
            "Paginated grid via limit/offset + X-Total-Count header",
            "Columns: created_at, action, target_type+target_id, admin_id",
            "redact_pii toggle → URL/query param (default true per legacy)",
            "Export CSV toolbar → GET .../audit/export?format=csv",
            "Show X-Export-Truncated + X-Next-Cursor banner when export truncated",
        ],
        out_scope=[
            "actor/action/date text filters (not in OpenAPI list op)",
            "ListEnvelope with total field in JSON body",
            "PII unmask for non-audit:read roles",
        ],
        not_on_page=["actor filter", "action filter", "date range filter", "invented total JSON field"],
        api_gaps=[
            (
                "List envelope",
                "auditList returns array + X-Total-Count header, not ListEnvelope",
                "Parse total from header in audit_api.ts; never invent body.total",
            ),
            (
                "actor/action/date filters",
                "Not on GET /api/v1/audit OpenAPI params",
                "No filter panel until backend + OpenAPI milestone",
            ),
            (
                "Export customer_id filter",
                "auditExport has optional customer_id; list does not",
                "Export-only param; do not add to list UI unless list op gains param",
            ),
            (
                "changes/metadata columns",
                "AuditLog has changes object — list UI shows summary only",
                "No expand row client-side PII parsing beyond redact_pii flag",
            ),
        ],
        stop_triggers=[
            "Invented filter UI without OpenAPI query params — stop",
            "Wrapper AuditDirectoryChrome — revert",
        ],
        contract_rows=[
            ("List op", "GET /api/v1/audit — auditList", "limit, offset, redact_pii"),
            ("List body", "AuditLog[]", "id, admin_id, action, target_type, target_id, created_at"),
            ("Total", "X-Total-Count response header", "integer — not JSON total"),
            ("Export", "GET /api/v1/audit/export — auditExport", "format=csv, redact_pii, cursor, customer_id"),
            ("Export headers", "X-Export-Truncated, X-Next-Cursor", "show truncation banner"),
            ("RBAC", "x-permissions", "audit:read"),
        ],
        regions=[
            Region("chrome", "PageChrome", 'Title "Audit log"; subtitle entry count', "X-Total-Count"),
            Region("toolbar", "AuditToolbar", "Export CSV button", "GET .../export"),
            Region("filters", "AuditFilterPanel", "redact_pii checkbox only", "URL redact_pii"),
            Region("content", "AuditGrid", "role=grid", "rows[]"),
            Region("col_time", "grid cell", "created_at locale", "row.created_at"),
            Region("col_action", "grid cell", "action string", "row.action"),
            Region("col_target", "grid cell", "target_type + target_id short", "row.target_*"),
            Region("col_admin", "grid cell", "admin_id short monospace", "row.admin_id"),
            Region("export_banner", "AlertBanner", "truncated export notice", "X-Export-Truncated"),
            Region("footer", "PaginationBar", "limit/offset pages", "header total"),
            Region("error", "ErrorBlock", "List/export failure", "fetch error"),
            Region("loading", "grid skeleton", "5 placeholder rows", "loading"),
            Region("empty", "EmptyState", "No audit entries copy", "rows.length === 0"),
        ],
        grid_cols="10rem 8rem minmax(10rem,1.2fr) 8rem",
        grid_ascii="""```
┌ PageChrome ─────────────────────────────────────────────┐
│ Audit log                         {total} entries       │
├ Toolbar ────────────────────────────────────────────────┤
│ [Export CSV]                                            │
├ FilterPanel ────────────────────────────────────────────┤
│ [x] Redact PII in changes/metadata                      │
├ Content (role=grid) ────────────────────────────────────┤
│ --audit-cols: 10rem 8rem minmax(10rem,1.2fr) 8rem       │
│ Time | Action | Target | Admin                          │
├ Footer ─────────────────────────────────────────────────┤
│ PaginationBar                                           │
└─────────────────────────────────────────────────────────┘
```""",
        grid_invariants=_dir_invariants("--audit-cols")
        + [
            ("Header total", "total from X-Total-Count only"),
            ("No actor/action columns sort", "No sort params in OpenAPI"),
        ],
        url_params=[
            UrlParam("limit", "limit", "50"),
            UrlParam("offset", "offset", "0"),
            UrlParam("redact_pii", "redact_pii", "true", "boolean; legacy default true"),
        ],
        sortable_cols=[],
        nonsortable_cols=["created_at", "action", "target", "admin_id"],
        fetch_example="GET /api/v1/audit?limit=50&offset=0&redact_pii=true",
        files=_default_dir_files("audit", "audit_page.tsx")
        + [
            ("web/src/helpers/api_blob.ts", "Export blob + truncation headers"),
            ("web/src/ui/audit/audit_export_banner.tsx", "Truncation notice"),
        ],
        legacy_remove=["../components/filter_toolbar.js", "<table> data-table", "page state page index without URL offset"],
        risks=[
            ("Invented total field", "Assumes ListEnvelope JSON", "Read X-Total-Count header"),
            ("Export without permission", "Show button without audit:read", "Gate toolbar"),
            ("Filter fiction", "actor/action UI without API", "api_gaps — no filters"),
            ("Silent export fail", "toast only", "Show truncation banner + error"),
        ],
        slop=[
            ("JSON total field", "body.total in helper", "Header parse only"),
            ("Date filter UI", "Not in OpenAPI", "Defer filter milestone"),
            ("Copy <table> layout", "Legacy audit_page", "role=grid subgrid"),
            ("Export success without download", "Missing blob handling", "api_blob.ts pattern"),
        ],
        laziness=[
            ("Keep page state pagination", "No URL offset", "Sync limit/offset in URL"),
            ("Skip export truncation UI", "Download only", "X-Export-Truncated banner"),
            ("Patch audit_page in place", "Smallest diff", "ui/audit/ sections"),
        ],
        forbidden=[
            '"Server-side actor filter" without OpenAPI param on auditList',
            '"total from response body" — use X-Total-Count',
            "Export button without audit:read permission",
        ],
        pitfalls=[
            ("Invented total field", "ListEnvelope lie — header only"),
            ("Actor/action filter UI", "Not in OpenAPI — blocked"),
            ("Silent catch → empty", "ErrorBlock on list fail"),
            ("Export without format=csv", "auditExport requires format enum csv"),
            ("Ignore export truncation", "Show X-Next-Cursor banner"),
            ("redact_pii not in URL", "Toggle must refetch with query param"),
            ("<table> instead of grid", "ui.mdc violation"),
            ("Wrapper stack", "No AuditDirectoryChrome"),
            ("Page-only pagination", "URL offset/limit sync"),
            ("PII expand client-side", "Respect redact_pii; no manual unmask"),
            ("Flex page layout", "CSS Grid sections"),
            ("Duplicate ErrorBlock", "Early return vs inline — one pattern"),
        ],
        impl_steps=[
            ("1", "api/openapi/paths/platform.yaml", "Confirm auditList + auditExport params", "openapi_gate"),
            ("2", "make openapi-types", "Regenerate openapi.d.ts", "typecheck"),
            ("3", "web/src/helpers/audit_api.ts", "listAudit parses X-Total-Count", "unit compile"),
            ("4", "web/src/ui/audit/*", "Grid + toolbar + redact toggle", "surface gate"),
            ("5", "web/src/pages/audit_page.tsx", "URL sync offset + compose", "no <table>"),
            ("6", "Export flow", "api_blob + truncation banner", "manual CSV download"),
            ("7", "web/src/app_routes.tsx", "Route /audit", "loads"),
            ("8", "Legacy cleanup", "Remove filter_toolbar/table", "rg data-table web/src/pages/audit_page.tsx empty"),
        ],
        sla_rows=_default_sla(),
        manual_checks=_default_manual("audit", "audit_page.tsx")
        + [
            ("Header total", "Manual: compare X-Total-Count vs row count", "total matches header"),
            ("redact_pii", "Manual: toggle off", "refetch with redact_pii=false"),
            ("Export", "Manual: Export CSV", "file downloads; truncation banner if header set"),
            ("Pagination", "Manual: offset in URL", "refetch new page"),
            ("Permission", "Manual: session without audit:read", "route hidden or forbidden"),
        ],
        extra_verify=["rg 'X-Total-Count' web/src/helpers/audit_api.ts"],
    ),
    MilestoneSpec(
        filename="ADMIN_DIRECTORY_MILESTONE_BILLING.md",
        slug="admin_directory_billing",
        title="ADMIN_DIRECTORY_MILESTONE_BILLING",
        depends=PAGE_CHROME_DEPS,
        blocks="`admin_detail_invoice`",
        pattern="admin_directory_pattern",
        summary="Operator invoice directory at /billing — KPI strip + invoice grid; legacy wallet/ledger tabs out of scope.",
        route="/billing",
        apis="GET /api/v1/billing/invoices — billingListInvoices; GET /api/v1/billing/summary — billingSummary",
        operation_id="billingListInvoices",
        schema_ref="InvoiceListResponse + BillingSummary — billing.yaml",
        handler="internal/billingadmin/handlers.go — billingListInvoices, billingSummary",
        domain="billing",
        legacy="billing_page.tsx",
        nav_group="Commercial → Billing",
        icon="receipt",
        permission="customers:read (invoices); shards:read (summary KPI)",
        in_scope=[
            "Rebuild invoice directory portion of `/billing` under `web/src/ui/billing/`",
            "Admin fleet invoice grid: id, customer_id, billing_month, total_micro, status, currency",
            "Optional KPI strip from GET /api/v1/billing/summary when session has shards:read",
            "URL filters: customer_id, status, month, min_total, limit, offset per OpenAPI",
            "Row → `/billing/invoices/{id}`",
            "Buyer bound customer: default customer_id filter from session",
        ],
        out_scope=[
            "Invoice detail (`ADMIN_DETAIL_MILESTONE_INVOICE.md`)",
            "Legacy wallet/ledger/exports/disputes tabs (separate milestones or detail)",
            "Client KPI math on invoice rows",
            "Crypto payment / self-serve billing panels",
            "RecentCustomers widget",
        ],
        not_on_page=["wallet tab", "ledger tab", "exports tab", "disputes tab", "client sortRows", "due_at column"],
        api_gaps=[
            (
                "due_at / amount_display",
                "Invoice/InvoiceSummary have total_micro not due_at or amount_display",
                "Format total_micro in UI; omit due column until OpenAPI field exists",
            ),
            (
                "status_label",
                "InvoiceSummary.status string only — no status_label/tone",
                "displayLabel(status) — no invented tone chip",
            ),
            (
                "billingSummary permission",
                "billingSummary requires shards:read not customers:read",
                "Hide KPI strip when permission missing — do not fake KPIs",
            ),
            (
                "Legacy multi-tab page",
                "billing_page.tsx has wallet/ledger/exports/disputes",
                "This milestone ships invoices directory + optional KPI only",
            ),
        ],
        stop_triggers=[
            "Ship all billing tabs in one PR — split per milestone",
            "Demo overdue KPI literals — revert",
        ],
        contract_rows=[
            ("List op", "GET /api/v1/billing/invoices — billingListInvoices", "customer_id, month, status, min_total, limit, offset"),
            ("List envelope", "InvoiceListResponse", "items, total, limit, offset"),
            ("Row", "Invoice / InvoiceSummary", "id, customer_id, billing_month, total_micro, status, currency"),
            ("Summary", "GET /api/v1/billing/summary — billingSummary", "invoiced_mtd_micro, invoice_count_mtd, undelivered_invoice_notifications"),
            ("RBAC list", "x-permissions", "customers:read"),
            ("RBAC summary", "x-permissions", "shards:read"),
        ],
        regions=[
            Region("chrome", "PageChrome", 'Title "Billing"', "static"),
            Region("kpis", "BillingSummaryStrip", "MTD invoiced, invoice count, undelivered notifications", "GET .../summary"),
            Region("filters", "BillingFilterPanel", "customer_id, status, month, min_total; Apply → URL", "URL params"),
            Region("content", "InvoicesGrid", "role=grid invoice columns", "items[]"),
            Region("col_id", "grid cell", "Invoice id link to detail", "item.id"),
            Region("col_customer", "grid cell", "customer_id short + link context", "item.customer_id"),
            Region("col_month", "grid cell", "billing_month", "item.billing_month"),
            Region("col_total", "grid cell", "formatAmountMicro(total_micro)", "item.total_micro"),
            Region("col_status", "grid cell", "status string", "item.status"),
            Region("footer", "PaginationBar", "limit/offset/total", "envelope"),
            Region("error", "ErrorBlock", "List/summary failure", "fetch error"),
            Region("loading", "grid skeleton", "placeholder rows", "loading"),
            Region("empty", "EmptyState", "No invoices copy", "items.length === 0"),
        ],
        grid_cols="10rem minmax(10rem,1.5fr) 7rem 8rem 7rem 5rem",
        grid_ascii="""```
┌ PageChrome ─────────────────────────────────────────────┐
│ Billing                                                 │
├ KPI strip (optional shards:read) ───────────────────────┤
│ Invoiced MTD | Invoice count | Undelivered notifications│
├ FilterPanel ────────────────────────────────────────────┤
│ Customer [____] Status [▼] Month [____] Min total [__]  │
├ Content (role=grid) ────────────────────────────────────┤
│ --billing-cols: 10rem minmax(10rem,1.5fr) 7rem 8rem 7rem 5rem │
│ ID | Customer | Month | Total | Status | Currency       │
├ Footer ─────────────────────────────────────────────────┤
│ PaginationBar                                           │
└─────────────────────────────────────────────────────────┘
```""",
        grid_invariants=_dir_invariants("--billing-cols")
        + [
            ("KPI strip optional", "Hidden without shards:read — no demo numbers"),
            ("No client sort", "No sort params on billingListInvoices"),
        ],
        url_params=[
            UrlParam("limit", "limit", "50"),
            UrlParam("offset", "offset", "0"),
            UrlParam("customer_id", "customer_id", "", "buyer session default"),
            UrlParam("status", "status", "", "invoice status filter"),
            UrlParam("month", "month", "", "MonthQuery YYYY-MM"),
            UrlParam("min_total", "min_total", "", "int64 micros floor"),
        ],
        sortable_cols=[],
        nonsortable_cols=["id", "customer", "month", "total", "status", "currency"],
        fetch_example="GET /api/v1/billing/invoices?limit=50&offset=0&customer_id={uuid}&status=open&month=2026-08",
        files=_default_dir_files("billing", "billing_page.tsx")
        + [
            ("web/src/ui/billing/billing_summary_strip.tsx", "KPI strip from billingSummary"),
            ("web/src/ui/billing/billing_summary_strip.module.css", "KPI grid"),
        ],
        legacy_remove=[
            "../components/billing_* panels from invoice route",
            "../lib/table_sort.js",
            "TabBar wallet/ledger/exports from directory milestone slice",
        ],
        risks=[
            ("Client KPI math", "Sum invoices in browser for KPI strip", "billingSummary endpoint only"),
            ("Demo overdue count", "Hardcoded KPI cards", "API summary fields or hide strip"),
            ("Ship all tabs", "Scope creep from legacy billing_page", "Invoices directory only"),
            ("sortRows on invoices", "Legacy table_sort", "No client sort"),
        ],
        slop=[
            ("due_at column", "Field not in OpenAPI", "Omit until schema ships"),
            ("KPI without shards:read", "Show strip with zeros", "Hide strip"),
            ("Wallet tab in same PR", "Monolith carryover", "Separate milestone"),
            ("amount_display field", "Invented DTO", "formatAmountMicro"),
        ],
        laziness=[
            ("Copy billing_page.tsx", "600-line monolith", "ui/billing invoices only"),
            ("Skip summary permission check", "Always fetch summary", "Gate on shards:read"),
            ("Keep TabBar", "wallet/ledger in page", "Drop tabs from this milestone"),
        ],
        forbidden=[
            '"Billing directory done" with wallet/ledger tabs still in same page compose',
            '"KPI strip wired" without shards:read or billingSummary 2xx',
            "due_at or amount_display columns without OpenAPI fields",
            "sortRows on items[]",
        ],
        pitfalls=[
            ("Client KPI math", "Use billingSummary only"),
            ("Demo overdue KPI", "No hardcoded numbers"),
            ("due_at column", "Not in InvoiceSummary schema"),
            ("status_label chip", "status string only — no tone API"),
            ("All tabs at once", "wallet/ledger out of scope"),
            ("Buyer scope leak", "Default customer_id for bound buyer"),
            ("sortRows legacy", "Remove table_sort"),
            ("RecentCustomers widget", "Not on directory page"),
            ("Summary 403 shown as zeros", "Hide KPI strip on error"),
            ("Flex page layout", "Grid sections"),
            ("Wrapper BillingDirectoryChrome", "Flat ui/billing/"),
            ("Silent catch → empty", "ErrorBlock on list fail"),
        ],
        impl_steps=[
            ("1", "api/openapi/paths/billing.yaml", "Confirm billingListInvoices + billingSummary", "openapi_gate"),
            ("2", "make openapi-types", "Regenerate openapi.d.ts", "typecheck"),
            ("3", "web/src/helpers/billing_api.ts", "listInvoices + getSummary helpers", "header/body parse only"),
            ("4", "web/src/ui/billing/*", "KPI strip + invoice grid sections", "surface gate"),
            ("5", "web/src/pages/billing_page.tsx", "Invoices compose + URL params", "no wallet TabBar in slice"),
            ("6", "web/src/app_routes.tsx", "Route /billing", "loads"),
            ("7", "Permission gates", "KPI strip shards:read; list customers:read", "manual role matrix"),
            ("8", "Legacy cleanup", "Remove table_sort from billing page", "rg table_sort web/src/pages/billing_page.tsx empty"),
        ],
        sla_rows=_default_sla(),
        manual_checks=_default_manual("billing", "billing_page.tsx")
        + [
            ("KPI permission", "Manual: session without shards:read", "KPI strip hidden"),
            ("Invoice row link", "Manual: click row", "navigates to /billing/invoices/{id}"),
            ("Filters", "Manual: customer_id in URL", "refetch filtered list"),
            ("No client sort", "rg table_sort web/src/ui/billing/", "no matches"),
            ("Error", "Manual: block API", "ErrorBlock visible"),
        ],
        extra_verify=["rg 'table_sort' web/src/pages/billing_page.tsx"],
    ),
    MilestoneSpec(
        filename="ADMIN_DIRECTORY_MILESTONE_RTB_DEALS.md",
        slug="admin_directory_rtb_deals",
        title="ADMIN_DIRECTORY_MILESTONE_RTB_DEALS",
        depends=PAGE_CHROME_DEPS,
        blocks="—",
        pattern="admin_directory_pattern",
        summary="RTB deals directory with inline create/edit modal; full array fetch (no pagination).",
        route="/rtb/deals",
        apis="GET/POST /api/v1/rtb/deals; PATCH/DELETE /api/v1/rtb/deals/{id}",
        operation_id="rtbListDeals",
        schema_ref="RtbDeal[] — rtb.yaml",
        handler="internal/rtbadmin/handlers.go — rtbListDeals, rtbCreateDeal, rtbPatchDeal, rtbDeleteDeal",
        domain="rtb",
        legacy="rtb_deals_page.tsx",
        nav_group="RTB → Deals",
        icon="handshake",
        permission="rtb:read (list); rtb:write (mutate)",
        in_scope=[
            "Rebuild `/rtb/deals` under `web/src/ui/rtb/`",
            "Grid: id, deal_id, floor_micro, customer_id, pacing, seats, updated_at",
            "Create/Edit modal with RtbDealCreateSpec fields from OpenAPI",
            "Delete row with confirm → DELETE .../rtb/deals/{id}",
            "Breadcrumb link back to /rtb/integration",
        ],
        out_scope=[
            "RTB integration profile (`ADMIN_DETAIL_MILESTONE_RTB.md`)",
            "Server pagination (rtbListDeals returns full array)",
            "geo_mask / cat_mask editing unless exposed in PATCH schema",
        ],
        not_on_page=["PaginationBar", "client sortRows", "501 stub forever on live handler"],
        api_gaps=[
            (
                "No pagination",
                "rtbListDeals returns RtbDeal[] without limit/offset",
                "No PaginationBar; full fetch; window if N>500",
            ),
            (
                "geo_mask / cat_mask",
                "On RtbDeal schema but not in legacy modal",
                "Omit columns until PATCH body documents editable masks",
            ),
            (
                "Path id type",
                "Deal path id is int64 not UUID",
                "Use deal.id (int64) for PATCH/DELETE paths",
            ),
        ],
        stop_triggers=[
            "Modal fields not on RtbDealCreateSpec — revert",
            "Wrapper RtbDealsChrome — revert",
        ],
        contract_rows=[
            ("List", "GET /api/v1/rtb/deals — rtbListDeals", "array of RtbDeal"),
            ("Create", "POST /api/v1/rtb/deals — rtbCreateDeal", "RtbDealCreateSpec body"),
            ("Update", "PATCH /api/v1/rtb/deals/{id} — rtbPatchDeal", "RtbDealCreateSpec body"),
            ("Delete", "DELETE /api/v1/rtb/deals/{id} — rtbDeleteDeal", "confirm modal"),
            ("Row", "RtbDeal", "id, deal_id, floor_micro, customer_id, pacing, seats, updated_at"),
            ("RBAC", "x-permissions", "rtb:read / rtb:write"),
        ],
        regions=[
            Region("chrome", "PageChrome", 'Title "RTB deals"', "deals.length"),
            Region("context", "ContextBar", "Breadcrumb RTB → Deals", "static routes"),
            Region("toolbar", "DealsToolbar", "Create deal button", "POST when rtb:write"),
            Region("content", "DealsGrid", "role=grid", "deals[]"),
            Region("col_deal_id", "grid cell", "deal_id monospace", "item.deal_id"),
            Region("col_floor", "grid cell", "formatAmountMicro(floor_micro)", "item.floor_micro"),
            Region("col_customer", "grid cell", "customer_id", "item.customer_id"),
            Region("col_pacing", "grid cell", "pacing label", "item.pacing"),
            Region("col_actions", "grid cell", "Edit / Delete", "rtb:write"),
            Region("modal", "DealFormModal", "Create/edit form", "RtbDealCreateSpec"),
            Region("error", "ErrorBlock", "List/save failure", "fetch error"),
            Region("loading", "grid skeleton", "placeholder rows", "loading"),
            Region("empty", "EmptyState", "No deals + create CTA", "deals.length === 0"),
        ],
        grid_cols="5rem minmax(8rem,1fr) 7rem minmax(10rem,1fr) 6rem 5rem 8rem",
        grid_ascii="""```
┌ PageChrome ─────────────────────────────────────────────┐
│ RTB deals                                               │
├ Toolbar ────────────────────────────────────────────────┤
│ [Create deal]                                           │
├ Content (role=grid) ────────────────────────────────────┤
│ --rtb-deals-cols: 5rem minmax(8rem,1fr) 7rem … 8rem      │
│ ID | Deal ID | Floor | Customer | Pacing | Seats | Updated│
├ (no pagination — full array) ───────────────────────────┤
└─────────────────────────────────────────────────────────┘
Modal: deal_id, customer_id, floor_micro, pacing, seats
```""",
        grid_invariants=_dir_invariants("--rtb-deals-cols")
        + [
            ("No pagination", "Full array from rtbListDeals"),
            ("int64 path id", "PATCH/DELETE use numeric id"),
        ],
        url_params=[],
        sortable_cols=[],
        nonsortable_cols=["deal_id", "floor", "customer", "pacing", "seats", "updated"],
        fetch_example="GET /api/v1/rtb/deals",
        files=[
            ("web/src/pages/rtb_deals_page.tsx", "Compose: fetch + modal state"),
            ("web/src/ui/rtb/deals_directory.tsx", "Section assembly"),
            ("web/src/ui/rtb/deals_grid.tsx", "role=grid + rows"),
            ("web/src/ui/rtb/deal_form_modal.tsx", "Create/edit form"),
            ("web/src/ui/rtb/deals_directory.module.css", "Page section grid"),
            ("web/src/ui/rtb/deals_grid.module.css", "Column template"),
            ("web/src/helpers/rtb_api.ts", "fetchRtbDeals, create, patch, delete"),
            ("web/src/types/generated/openapi.d.ts", "Generated types"),
        ],
        legacy_remove=["../components/modal.js", "../components/breadcrumbs.js", "<table> data-table"],
        risks=[
            ("UUID path for deal", "Wrong id type on PATCH", "Use int64 id from row"),
            ("501 treated as empty", "Legacy stub path hides errors", "ErrorBlock unless true 501 stub route"),
            ("Modal fields invented", "geo_mask in form without API", "RtbDealCreateSpec only"),
            ("Delete without confirm", "One-click delete", "Confirm modal"),
        ],
        slop=[
            ("PaginationBar", "No list envelope", "Omit footer"),
            ("Flex page header", "page-header__row flex", "Grid sections"),
            ("settings:write fallback", "Legacy uses settings:write for mutate", "Prefer rtb:write per OpenAPI"),
            ("Copy monolith page", "300-line rtb_deals_page", "ui/rtb/ sections"),
        ],
        laziness=[
            ("Patch rtb_deals_page in place", "Keep Modal in page", "ui/rtb/deal_form_modal"),
            ("Skip delete flow", "Edit only", "DELETE wired with confirm"),
            ("Skip int64 id typing", "String id in URL", "OpenAPI int64 path"),
        ],
        forbidden=[
            '"Paginated deals list" without OpenAPI limit/offset',
            "Form fields not on RtbDealCreateSpec",
            "Delete without confirm modal",
        ],
        pitfalls=[
            ("Wrong path id type", "int64 not UUID on PATCH/DELETE"),
            ("Pagination fiction", "No envelope — no PaginationBar"),
            ("501 stub as success", "Show ErrorBlock on real errors"),
            ("Modal fields invented", "OpenAPI body only"),
            ("Delete no confirm", "Confirm modal required"),
            ("Flex page layout", "Grid sections"),
            ("<table> grid", "role=grid subgrid"),
            ("Toast before save 2xx", "apiConfirmed"),
            ("settings:write only gate", "Document rtb:write primary"),
            ("geo_mask column", "Not in legacy modal — omit"),
            ("Wrapper stack", "No RtbDealsChrome"),
            ("Silent catch → empty", "ErrorBlock on load fail"),
        ],
        impl_steps=[
            ("1", "api/openapi/paths/rtb.yaml", "Confirm deal CRUD ops + RtbDealCreateSpec", "openapi_gate"),
            ("2", "make openapi-types", "Regenerate openapi.d.ts", "typecheck"),
            ("3", "web/src/helpers/rtb_api.ts", "list/create/patch/delete typed", "int64 id paths"),
            ("4", "web/src/ui/rtb/*", "Grid + modal sections", "surface gate"),
            ("5", "web/src/pages/rtb_deals_page.tsx", "Thin compose", "no <table>"),
            ("6", "web/src/app_routes.tsx", "Route /rtb/deals", "loads"),
            ("7", "Delete confirm", "Confirm modal before DELETE", "manual test"),
            ("8", "Legacy cleanup", "Remove components/modal import", "rg '../components/modal' web/src/pages/rtb_deals_page.tsx empty"),
        ],
        sla_rows=_default_sla(),
        manual_checks=_default_manual("rtb", "rtb_deals_page.tsx")
        + [
            ("Create", "Manual: create deal", "POST 201; grid refetch"),
            ("Edit", "Manual: patch deal", "PATCH 2xx"),
            ("Delete", "Manual: delete with confirm", "DELETE 2xx; row removed"),
            ("No pagination", "rg PaginationBar web/src/ui/rtb/", "no matches"),
            ("Error", "Manual: block API", "ErrorBlock visible"),
        ],
    ),
    MilestoneSpec(
        filename="ADMIN_DIRECTORY_MILESTONE_BRANDS.md",
        slug="admin_directory_brands",
        title="ADMIN_DIRECTORY_MILESTONE_BRANDS",
        depends=PAGE_CHROME_DEPS,
        blocks="—",
        pattern="admin_directory_pattern",
        summary="Brands directory (route gap) — customer-scoped brand grid + create; creatives sub-panel per API.",
        route="/brands",
        apis="GET/POST /api/v1/brands; GET/POST /api/v1/brands/{id}/creatives",
        operation_id="brandsList",
        schema_ref="Brand[] + BrandCreative[] — campaign.yaml",
        handler="internal/brand/handlers.go — brandsList, brandsCreate, brandCreativesList",
        domain="brands",
        legacy="gap — no legacy page",
        nav_group="Commercial → Brands (new)",
        icon="tag",
        permission="campaigns:read (list); campaigns:write (create)",
        gap_route=True,
        in_scope=[
            "Register `/brands` route + nav entry",
            "Customer scope: required customer_id query (OpenAPI CustomerIdQueryRequired)",
            "Brand grid: name, id, updated_at, freq_limit/freq_window",
            "Create brand modal → POST /api/v1/brands",
            "Row expand or side panel: creatives list GET .../brands/{id}/creatives",
            "Create creative when campaigns:write",
        ],
        out_scope=[
            "Campaign brand picker embedded in campaign editor",
            "Server pagination (brandsList returns array)",
            "Global brand list without customer_id",
        ],
        not_on_page=["PaginationBar", "brand list without customer_id", "client sortRows"],
        api_gaps=[
            (
                "Route gap",
                "No /brands in legacy app_routes or pages",
                "Register route + nav before live:true",
            ),
            (
                "customer_id required",
                "brandsList requires customer_id query param",
                "FilterPanel or session default — cannot list all brands fleet-wide",
            ),
            (
                "No ListEnvelope",
                "brandsList returns Brand[] array",
                "No PaginationBar; full fetch per customer",
            ),
            (
                "Creatives no pagination",
                "brandCreativesList returns array",
                "Sub-panel lists all creatives for brand",
            ),
        ],
        stop_triggers=[
            "live:true before route registered — revert",
            "Brand grid without customer_id — API will 400",
        ],
        contract_rows=[
            ("List brands", "GET /api/v1/brands — brandsList", "customer_id required query"),
            ("Create brand", "POST /api/v1/brands — brandsCreate", "CreateBrandRequest"),
            ("List creatives", "GET /api/v1/brands/{id}/creatives — brandCreativesList", "BrandCreative[]"),
            ("Create creative", "POST /api/v1/brands/{id}/creatives — brandCreativesCreate", "create body"),
            ("Row", "Brand", "id, customer_id, name, freq_limit, freq_window, updated_at"),
            ("RBAC", "x-permissions", "campaigns:read / campaigns:write"),
        ],
        regions=[
            Region("chrome", "PageChrome", 'Title "Brands"', "brands.length"),
            Region("filters", "BrandsFilterPanel", "customer_id required; Apply → URL", "customer_id query"),
            Region("toolbar", "BrandsToolbar", "Create brand", "POST /brands"),
            Region("content", "BrandsGrid", "role=grid brand rows", "brands[]"),
            Region("col_name", "grid cell", "brand name", "item.name"),
            Region("col_id", "grid cell", "CopyableUuid id", "item.id"),
            Region("col_updated", "grid cell", "updated_at locale", "item.updated_at"),
            Region("col_freq", "grid cell", "freq_limit / freq_window", "item.freq_*"),
            Region("creatives", "BrandCreativesPanel", "expand row creatives grid", "GET .../creatives"),
            Region("modal", "BrandCreateModal", "name + customer_id", "CreateBrandRequest"),
            Region("error", "ErrorBlock", "List/create failure", "errors"),
            Region("loading", "grid skeleton", "placeholder rows", "loading"),
            Region("empty", "EmptyState", "No brands for customer", "array empty"),
        ],
        grid_cols="minmax(12rem,2fr) 10rem 8rem 7rem",
        grid_ascii="""```
┌ PageChrome ─────────────────────────────────────────────┐
│ Brands                                                  │
├ FilterPanel ────────────────────────────────────────────┤
│ Customer ID [___________] [Apply]  (required)           │
├ Toolbar ────────────────────────────────────────────────┤
│ [Create brand]                                          │
├ Content (role=grid) ────────────────────────────────────┤
│ --brands-cols: minmax(12rem,2fr) 10rem 8rem 7rem        │
│ Name | ID | Updated | Freq cap                          │
├ Creatives panel (expanded row) ─────────────────────────┤
│ Creative name | landing_url | weight | status           │
└─────────────────────────────────────────────────────────┘
```""",
        grid_invariants=_dir_invariants("--brands-cols")
        + [
            ("customer_id required", "No fetch without customer_id in URL"),
            ("No pagination", "Array response — no PaginationBar"),
        ],
        url_params=[
            UrlParam("customer_id", "customer_id", "", "required — CustomerIdQueryRequired"),
        ],
        sortable_cols=[],
        nonsortable_cols=["name", "id", "updated_at", "freq"],
        fetch_example="GET /api/v1/brands?customer_id={uuid}",
        files=_default_dir_files("brands", "brands_page.tsx")
        + [
            ("web/src/ui/brands/brand_creatives_panel.tsx", "Creatives sub-grid"),
            ("web/src/ui/brands/brand_create_modal.tsx", "Create brand form"),
            ("web/src/helpers/nav_config.ts", "Add /brands nav entry"),
        ],
        legacy_remove=["N/A — new route"],
        risks=[
            ("live without route", "Catalog live before app_routes", "Register route first"),
            ("Missing customer_id", "brandsList 400", "URL required param"),
            ("Fleet-wide brand list", "API requires customer scope", "No admin-all brands until API exists"),
        ],
        slop=[
            ("live:true early", "Nav link 404", "gap_route until route ships"),
            ("PaginationBar", "No list envelope", "Omit"),
            ("Creatives without brand row", "Orphan panel", "Expand selected brand only"),
        ],
        laziness=[
            ("Skip nav_config", "Route only", "Nav + permission in same PR"),
            ("Skip creatives panel", "Brands only", "Milestone includes creatives sub-route"),
            ("Hand-written Brand type", "types/brand.js", "openapi.d.ts"),
        ],
        forbidden=[
            '"Brands directory live" without /brands in app_routes.tsx',
            "List fetch without customer_id query param",
            "Pagination without OpenAPI limit/offset",
        ],
        pitfalls=[
            ("live without route", "Register app_routes + nav first"),
            ("Missing customer_id", "API 400 — block fetch until set"),
            ("Fleet-wide list fiction", "customer_id required in OpenAPI"),
            ("PaginationBar", "Array response — omit"),
            ("Wrapper BrandsDirectoryChrome", "Flat ui/brands/"),
            ("Campaign picker scope", "Embedded picker is different surface"),
            ("Creative fields invented", "BrandCreative schema only"),
            ("Toast before create 2xx", "apiConfirmed"),
            ("Flex page layout", "Grid sections"),
            ("Silent catch → empty", "ErrorBlock"),
            ("Skip creatives API", "brandCreativesList wired on expand"),
            ("Wrong permission slug", "campaigns:read not brands:read"),
        ],
        impl_steps=[
            ("1", "web/src/app_routes.tsx", "Add /brands route", "Route resolves"),
            ("2", "web/src/helpers/nav_config.ts", "Nav entry campaigns:read", "visible with permission"),
            ("3", "api/openapi/paths/campaigns.yaml", "Confirm brands + creatives ops", "openapi_gate"),
            ("4", "make openapi-types", "Regenerate openapi.d.ts", "typecheck"),
            ("5", "web/src/helpers/brands_api.ts", "list/create + creatives helpers", "customer_id param"),
            ("6", "web/src/ui/brands/*", "Grid + modal + creatives panel", "surface gate"),
            ("7", "web/src/pages/brands_page.tsx", "URL customer_id compose", "no fetch without id"),
            ("8", "Set live flag honest", "nav + route + handler", "report_live_routes_gate"),
        ],
        sla_rows=_default_sla(),
        manual_checks=_default_manual("brands", "brands_page.tsx")
        + [
            ("Route", "Manual: open /brands", "page loads not 404"),
            ("customer_id", "Manual: missing param", "no fetch / validation message"),
            ("Create brand", "Manual: POST create", "201 + grid refetch"),
            ("Creatives", "Manual: expand row", "creatives list loads"),
            ("Nav", "Manual: nav shows Brands", "permission gated"),
        ],
    ),
]


def _detail_files(domain: str, page: str) -> list[tuple[str, str]]:
    return [
        (f"web/src/pages/{page}", "Thin compose; tabs route optional"),
        (f"web/src/ui/{domain}/{domain}_detail.tsx", "Detail shell"),
        (f"web/src/ui/{domain}/*.module.css", "Section CSS modules"),
        (f"web/src/helpers/{domain}_api.ts", "get/patch + sub-resource fetches"),
        ("web/src/types/generated/openapi.d.ts", "Generated types"),
    ]
