export type DocsTopic = {
  problem: string;
  symptom: string;
  fix: string;
};

export type DocsSection = {
  id: string;
  title: string;
  summary: string;
  topics: DocsTopic[];
};

export const DOCS_SECTIONS: DocsSection[] = [
  {
    id: 'login-session',
    title: 'Login & session',
    summary: 'Sign-in, cookies, CSRF, and session bootstrap failures.',
    topics: [
      {
        problem: 'Login returns 401 or loops back to sign-in',
        symptom: 'Credentials look correct but you land on /login again.',
        fix: 'Re-run seed: bash scripts/dev/stack/seed_admin.sh. Dev login: admin@test.local / Password123!. Hard-refresh after login.',
      },
      {
        problem: 'Session bootstrap 404',
        symptom: 'Network tab shows GET /api/v1/session/bootstrap as 404; UI may still work via fallback.',
        fix: 'Rebuild control image: bash scripts/dev/stack/stack.sh build && bash scripts/dev/stack/stack.sh ingest-only. Or use web dev proxy (cd web && npm run dev) against :8188.',
      },
      {
        problem: 'CSRF or 403 on save',
        symptom: 'PATCH/POST fails with forbidden or CSRF after idle tab.',
        fix: 'Hard refresh (Ctrl+F5). Confirm GET /api/v1/auth/me returns 200 before writes.',
      },
      {
        problem: '403 Forbidden page',
        symptom: 'Redirect to /forbidden after login.',
        fix: 'Check permissions on /api/v1/auth/me. Ops routes need operator access; seed admin has full ACL in dev.',
      },
    ],
  },
  {
    id: 'license-setup',
    title: 'License & bootstrap',
    summary: 'First-run setup, EULA gate, and license tier limits.',
    topics: [
      {
        problem: 'Stuck on /setup',
        symptom: 'Bootstrap never completes.',
        fix: 'Open GET /api/v1/meta — bootstrap_complete must be true. Finish platform bootstrap in Settings or run stack seed.',
      },
      {
        problem: 'EULA modal blocks navigation',
        symptom: 'Dialog on every page until accepted.',
        fix: 'Accept EULA in the gate dialog. If text is empty, check /api/v1/meta eula fields and control logs.',
      },
      {
        problem: 'License setup screen',
        symptom: 'Forced license page after login.',
        fix: 'Install JWT to var/license.jwt per docs/DEVELOPMENT.md. Settings -> License shows current tier and limits.',
      },
      {
        problem: 'Feature missing or 501 stub',
        symptom: 'StubBanner or HTTP 501 on a page.',
        fix: 'SKU may not include the feature. Check license entitlements; upgrade tier or use a dev license with the flag enabled.',
      },
    ],
  },
  {
    id: 'campaigns',
    title: 'Campaigns & traffic',
    summary: 'Editor, publish, budgets, and empty campaign lists.',
    topics: [
      {
        problem: 'Campaign list empty',
        symptom: 'Table has no rows after Apply.',
        fix: 'Set Customer ID filter to a real UUID from Customers. Clear status filter. Confirm GET /api/v1/campaigns returns items.',
      },
      {
        problem: 'Publish or save fails',
        symptom: 'Toast error or 4xx on PATCH.',
        fix: 'Open campaign editor validation banner. Check budget > 0, required integrations, and campaigns:write permission.',
      },
      {
        problem: 'Spend not moving',
        symptom: 'Pacing frozen while traffic runs.',
        fix: 'Verify tracker ingest is up (stack ingest-only). Ops -> Metrics for lag. Budget debits live in Redis; PG spend syncs async.',
      },
      {
        problem: 'Import or wizard stuck',
        symptom: 'Poll never finishes in campaign wizard.',
        fix: 'Open Manage session dialog and retry commit. Check Ops -> Outbox for stuck jobs; DLQ for failed import payloads.',
      },
    ],
  },
  {
    id: 'billing',
    title: 'Billing & invoices',
    summary: 'Invoices, exports, and customer billing views.',
    topics: [
      {
        problem: 'Invoice list empty',
        symptom: 'No rows for a known customer.',
        fix: 'Pick customer in Billing filters. Date range may exclude drafts. Confirm customer_id matches Postgres billing records.',
      },
      {
        problem: 'Export download fails',
        symptom: 'Export job errors or empty file.',
        fix: 'Narrow From/To range. Check Reports/Billing exports job status. Ops -> DLQ for failed export workers.',
      },
      {
        problem: 'Totals mismatch',
        symptom: 'Invoice lines do not match dashboard spend.',
        fix: 'Billing is batch-oriented; allow recon window. Ops -> Recon for shard/settlement drift. Not a hot-path real-time view.',
      },
    ],
  },
  {
    id: 'integrations',
    title: 'Integrations & postbacks',
    summary: 'Cost sync, schemas, affiliate presets, and postback DLQ.',
    topics: [
      {
        problem: 'Cost sync credential rejected',
        symptom: 'PUT credentials returns 400/422.',
        fix: 'Match OpenAPI schema in Integrations -> Cost sync. Network slug must be lowercase; secrets are write-only on save.',
      },
      {
        problem: 'Postback retries exhausted',
        symptom: 'Rows in Integrations -> Postbacks DLQ.',
        fix: 'Inspect payload and response code. Fix endpoint URL or auth, then retry from DLQ inbox (Ops -> DLQ).',
      },
      {
        problem: 'Schema test connection fails',
        symptom: 'Integration hub shows error on probe.',
        fix: 'Validate JSON mapping against sample payload. Check outbound network from control container to partner API.',
      },
    ],
  },
  {
    id: 'fraud',
    title: 'Fraud & traffic quality',
    summary: 'Labels, presets, silent reject, and decision overrides.',
    topics: [
      {
        problem: 'Label change not affecting traffic',
        symptom: 'Fraud label saved but blocks unchanged.',
        fix: 'Labels apply on next scoring batch or edge snapshot refresh. Check Fraud -> Integrations sync status; not instant on hot path.',
      },
      {
        problem: 'Silent reject confusion',
        symptom: 'Events accepted with 202 but analytics differ.',
        fix: 'Silent reject is per-IP decoy, not campaign flag toggle alone. Verify silent_reject_event in CH funnels, not legacy ghost_* columns.',
      },
      {
        problem: 'Preset patch no effect',
        symptom: 'Threshold change does not move block rate.',
        fix: 'Confirm preset bound to campaign fraud panel. ML scoring is batch-sidecar; boost snapshot updates async.',
      },
    ],
  },
  {
    id: 'ops',
    title: 'Ops & stack health',
    summary: 'DLQ, outbox, shards, blacklist, and local compose.',
    topics: [
      {
        problem: 'Stack services down',
        symptom: 'API connection refused on :8188.',
        fix: 'bash scripts/dev/stack/stack.sh ingest-only after build. docker compose ps under deploy/compose. Logs: docker compose logs control.',
      },
      {
        problem: 'DLQ growing',
        symptom: 'Ops -> DLQ inbox count rising.',
        fix: 'Open oldest message, fix root cause (schema, auth, downstream 5xx), replay or drop after fix.',
      },
      {
        problem: 'Outbox lag',
        symptom: 'Ops -> Outbox pending age high.',
        fix: 'Check Redis and worker health. Shard 0 catchup may be running after restart; wait or inspect Ops -> Shards.',
      },
      {
        problem: 'Roles reload needed',
        symptom: 'New permissions not visible after team change.',
        fix: 'Ops -> Home -> Reload roles, then hard refresh browser session.',
      },
    ],
  },
  {
    id: 'reports',
    title: 'Reports & dashboards',
    summary: 'Role dashboards, report jobs, and stale metrics.',
    topics: [
      {
        problem: 'Dashboard Load disabled',
        symptom: 'Button greyed out on dashboards.',
        fix: 'Customer ID is required. Set From/To range; pick role (Buyer, AdOps, etc.) and Apply.',
      },
      {
        problem: 'Stale dashboard numbers',
        symptom: 'Metrics old after traffic spike.',
        fix: 'Hard refresh. CH rollups lag minutes, not seconds. Compare with Ops -> Metrics ingest lag.',
      },
      {
        problem: 'Report job failed',
        symptom: 'Reports -> Jobs shows error state.',
        fix: 'Open job detail for SQL/timeout message. Narrow date range; heavy reports are cold-path only.',
      },
    ],
  },
  {
    id: 'local-dev',
    title: 'Local dev quick fixes',
    summary: 'Admin UI dev server, fonts, and common compose mistakes.',
    topics: [
      {
        problem: 'Admin UI dev blank or 404 assets',
        symptom: 'White page or missing fonts on :5173.',
        fix: 'cd web && npm run dev. API proxies to :8188. Run npm run build if testing embed bundle.',
      },
      {
        problem: 'Typecheck or OpenAPI drift',
        symptom: 'CI fails on admin web gate.',
        fix: 'make openapi-types after spec change. cd web && npm run typecheck. bash scripts/ci/admin/web.sh before push.',
      },
      {
        problem: 'OOM on full go test',
        symptom: 'Dev machine swaps on compile.',
        fix: 'Use make test-fast or scoped go test. Full integration: make test-integration when infra is up.',
      },
      {
        problem: 'Wrong stack profile',
        symptom: 'ClickHouse or Redis missing.',
        fix: 'ingest-only for tracker work; full for analytics-ml. See docs/DEVELOPMENT.md stack profile table.',
      },
    ],
  },
];

export const DEFAULT_DOCS_SECTION_ID = DOCS_SECTIONS[0]?.id ?? 'login-session';

export function getDocsSection(id: string | undefined): DocsSection | undefined {
  if (!id) {
    return undefined;
  }
  return DOCS_SECTIONS.find((section) => section.id === id);
}

export function isDocsSectionId(id: string): boolean {
  return DOCS_SECTIONS.some((section) => section.id === id);
}
