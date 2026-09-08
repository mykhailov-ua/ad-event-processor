import type { DocsGuide, DocsTopic } from '@/lib/docs_types';

export type { DocsTopic };

export type DocsSection = {
  id: string;
  title: string;
  summary: string;
  topics?: DocsTopic[];
  guides?: DocsGuide[];
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
        symptom:
          'Network tab shows GET /api/v1/session/bootstrap as 404; UI may still work via fallback.',
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
        problem: 'Stuck before sign-in',
        symptom: 'Redirected away from the console on first visit.',
        fix: 'Open /activate, paste license JWT, create owner email and password. GET /api/v1/meta should show bootstrap_complete=true after success.',
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
    summary: 'Labels, presets, non-blocking fraud responses, and decision overrides.',
    topics: [
      {
        problem: 'Label change not affecting traffic',
        symptom: 'Fraud label saved but blocks unchanged.',
        fix: 'Labels apply on next scoring batch or edge snapshot refresh. Check Fraud -> Integrations sync status; not instant on hot path.',
      },
      {
        problem: 'Non-blocking response analytics mismatch',
        symptom: 'Events accepted with 202 but analytics differ.',
        fix: 'Non-blocking fraud response is per-IP policy routing, not the campaign toggle alone. Verify silent_reject_event in ClickHouse funnels, not legacy analytics column names.',
      },
      {
        problem: 'Preset patch no effect',
        symptom: 'Threshold change does not move block rate.',
        fix: 'Confirm preset bound to campaign fraud panel. ML scoring runs in cmd/fraud-scorer batch only; tracker reads ml:score:boost snapshot async, not inline inference on /track.',
      },
      {
        problem: 'Safe-page bypass expectation',
        symptom: 'Moderator still reaches offer despite attestation.',
        fix: 'Read Documentation -> Fraud signal limits. Residential egress + real browser may pass single-layer checks; enable cross_layer_desync_action when stacked mismatches should route to safe page.',
      },
    ],
  },
  {
    id: 'fraud-signal-limits',
    title: 'Fraud signal limits',
    summary:
      'Honest matrix of ingress, safe-page, residential proxy, and ML enforcement limits. Canonical operator copy: deploy/vendor/ANTIFRAUD.md.',
    guides: [
      {
        id: 'signal-matrix',
        title: 'Signal matrix',
        blocks: [
          {
            type: 'note',
            text: 'Residential crawler on datacenter egress is detectable when edge headers are present. Residential crawler on residential egress is not fully detectable at L4 alone.',
          },
          {
            type: 'table',
            headers: ['Layer', 'Signal', 'Evades when'],
            rows: [
              ['L4 XDP', 'Host map / flood drop', 'Rotating residential IP, CDN front door'],
              [
                'TCP ingress',
                'TTL/window vs UA',
                'CDN path; OS_FINGERPRINT_MISMATCH_ENABLED=false',
              ],
              [
                'TLS ingress',
                'JA3/JA4 blocklist + corpus',
                'TLS terminated at CDN (headers missing)',
              ],
              [
                'Safe-page JS',
                'Canvas/WebGL/timezone attestation',
                'safe_page_enabled=false or real mobile browser',
              ],
              [
                'Mobile biometrics',
                'Gyro/touch on safe-page verify (click when enabled)',
                'MOBILE_BIOMETRICS_CLICK_ENABLED=0 or attestation off',
              ],
              [
                'Cross-layer desync',
                'Campaign cross_layer_desync_action (boost/safe_page/block)',
                'Residential IP may pass individual L2 signals; needs stacked mismatches',
              ],
              [
                'CGNAT collateral',
                'Mobile carrier blacklist bypass',
                'Probe cluster corroboration still hard-routes',
              ],
              [
                'Apple Private Relay',
                'DCASN exempt for relay ASN + Apple UA',
                'Non-Apple UA on relay ASN still datacenter_ip',
              ],
              [
                'In-app WebView',
                'Sec-Fetch/JA4 relax + social_in_app preset',
                'Desktop Chrome spoofing WebView substring',
              ],
              [
                'ML boost',
                'Redis snapshot on /track',
                'Not inline LGBM; cmd/fraud-scorer batch + outbox only',
              ],
              [
                'ResidentialProxyFilter',
                'Farm/heuristic intel',
                'Clean residential IP; not moderator proof',
              ],
            ],
          },
        ],
      },
      {
        id: 'ml-positioning',
        title: 'ML enforcement path',
        blocks: [
          {
            type: 'paragraph',
            text: 'Tracker FilterEngine adds a fraud score boost from an in-memory snapshot (SettingsWatcher). LightGBM inference runs only in cmd/fraud-scorer or embedded ivt-detector batch workers. Suspect-tier scores enqueue ML_SCORE_BOOST outbox rows; they do not run synchronously on POST /track.',
          },
          {
            type: 'list',
            items: [
              'Campaign fraud panel shows ml_boost_last_refreshed_at when Redis boost key is present.',
              'No admin UI or comment should imply per-request model inference on the hot path.',
              'SKU: ml_fraud_boost gates the fraud-scorer sidecar.',
            ],
          },
        ],
      },
    ],
    topics: [
      {
        problem: 'Buyer expects WebGL bypass on every click',
        symptom: 'Safe-page enabled but moderator reaches offer.',
        fix: 'Attestation is one layer. Check review_traffic_action, TLS corpus, and cross-layer reports. Enable multiple signals; read ANTIFRAUD.md safe-page section.',
      },
      {
        problem: 'XDP should block residential crawlers',
        symptom: 'Moderator on residential IP passes edge.',
        fix: 'XDP is flood + blocklist, not cloaking detection. Use ResidentialProxyFilter heuristics + review corpus; expect fail-open on unknown residential egress.',
      },
    ],
  },
  {
    id: 'perimeter-sybil-controls',
    title: 'Perimeter Sybil controls (T2)',
    summary:
      'Human-in-the-loop operators on residential cellular: operational limits, wave response, and honest buyer copy. Canonical: deploy/vendor/PERIMETER_INTEL_DEFENSE.md section T2.',
    guides: [
      {
        id: 'sybil-limits',
        title: 'What ingress cannot block',
        blocks: [
          {
            type: 'note',
            text: 'Real mobile device + LTE egress + human SOP may pass safe-page attestation. Automated threat intel feeds may not list the session. Operational and contractual controls are required.',
          },
          {
            type: 'list',
            items: [
              'Do not promise buyers "blocks all scrapers" or guaranteed moderator block.',
              'When crowd wave alerts fire, investigate before widening production routing.',
              'Do not bulk-promote residential/mobile IPs into XDP deny maps (CGNAT collateral).',
              'Separate authorized audit windows (known ASN/IP allowlist) from open production traffic.',
            ],
          },
        ],
      },
      {
        id: 'sybil-signals',
        title: 'Signals to monitor',
        blocks: [
          {
            type: 'table',
            headers: ['Signal', 'Surface'],
            rows: [
              ['ad_hybrid_crowd_wave_total', 'Prometheus'],
              ['GET /api/v1/fraud/crowd-waves/{campaign_id}', 'Admin cold API'],
              ['crowd_probe_score, crowd_wave_active', 'ClickHouse click rows'],
              ['Safe-page verify rate vs clicks', 'Reports / CH drill'],
            ],
          },
        ],
      },
    ],
    topics: [
      {
        problem: 'Moderator still reaches offer',
        symptom: 'Human on phone passes attestation despite safe-page enabled.',
        fix: 'Expected for clean Sybil sessions. Enable crowd wave + cross-layer policy; read PERIMETER_INTEL_DEFENSE.md T2 response playbook. Escalate via contract, not `/24` IP ban alone.',
      },
      {
        problem: 'Crowd wave alert during traffic spike',
        symptom: 'Wave score high after partner send or geo shift.',
        fix: 'Confirm organic burst before changing presets. Keep review_traffic_action=safe_page until investigated.',
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
