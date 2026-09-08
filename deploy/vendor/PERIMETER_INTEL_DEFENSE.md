# Perimeter intel defense

Operator and Blue Team reference for **Public Safe Sandbox Zone** vs **Internal Trusted Production Zone** separation, coordinated probe defense, and honest limits of ingress-side automation.

Cross-ref:

- `deploy/vendor/ANTIFRAUD.md` — fraud signal semantics, filter chain, ML batch path
- `ARBITRAGE_CLOSURE_BACKLOG.md` — remediation slugs (P3/P5) and delivery waves
- `.cursor/rules/edge.mdc` — TLS server persona (T15), edge Lua pipeline

Ingress routing and attestation are **not** a substitute for contractual access governance, moderator contracts, or production credential policy.

---

## Zone model

| Zone | What the user sees | Server gate |
| :--- | :--- | :--- |
| Public Safe Sandbox | Safe-page stub, decoy shell, attestation verify | `safe_page_enabled`, `review_traffic_policy`, crowd/probe filters |
| Internal Trusted Production | Money lander / offer redirect | Server verify verdict, budget debit, filter pass |

Unauthorized third-party scanners, human Sybil operators on residential cellular, and vision agents may still map zone boundaries when signals are clean and operational controls are missing.

---

## Threat catalog (T1–T30)

| ID | Threat | Severity | Backlog slug | Shipped mitigations (partial) |
| :--- | :--- | :--- | :--- | :--- |
| T1 | Dual-origin fetch: sandbox HTML differs from production | Medium | P5-DECOY-LANDING-PARITY, P5-CONTENT-DIFF-DRILL | Decoy parity, content-diff drill |
| T2 | Human-in-the-loop Sybil on residential cellular | High | P5-SYBIL-HUMAN-OPERATOR-RUNBOOK | Crowd wave + probe cluster; **operational controls required** |
| T3 | Empty sandbox hydrator (attestation no-op) | Low | P5-SAFE-PAGE-HYDRATOR-CLIENT | Hydrator client shipped |
| T4 | Static AST / obvious telemetry exports | Medium | P5-CLIENT-TELEMETRY-STEALTH-PACKAGING | Stealth bundle when `attestation_mode=strict` |
| T5 | Redirect chain fingerprinting | Medium | P5-REDIRECT-PROFILE-COMPLIANCE | Macro / DMR policy |
| T6 | (Reserved) | — | — | — |
| T7 | (Reserved) | — | — | — |
| T8 | Client integrity bypass on attestation API | Medium | P5-HYBRID-SERVER-VERIFY-GATE | Server-authoritative verify |
| T9 | (Reserved) | — | — | — |
| T10 | Threat intel corpus false positive | Operator | P3-MODERATOR-FINGERPRINT-CORPUS | Corpus QA, tuning |
| T11 | Coordinated probe-operator micro-behavior | Medium | P5-CROWD-PROBE-SCORING, P5-BEHAVIOR-MODEL-HUMANIZATION | Crowd probe scoring |
| T12 | Cross-session device reuse after storage wipe | Medium | P5-PROBE-CLUSTER-GRAPH, P5-ASN-MOBILE-TIER | Cluster graph + mobile ASN tier |
| T13 | Hybrid AI + crowd wave reconnaissance | Medium | P5-HYBRID-CROWD-WAVE-DETECTION | Wave detection + decoy gate |
| T14 | Client-edge DOM / on-device ML inference | High | P5-CLIENT-EDGE-DOM-INTEGRITY | DOM lint, CSP; **no server CV on hot path** |
| T15 | TLS server persona / H2 interrogation | Medium | P5-TLS-SERVER-PERSONA-HARDENING | Mozilla Intermediate cipher profile |
| T16 | Routing timing side-channel (TTFB by route class) | Medium | P5-ROUTING-TIMING-CONSTANT-TIME | Timing pad (backlog) |
| T17 | Hot-path telemetry heap allocs | Medium | P5-HOTPATH-TELEMETRY-ZERO-ALLOC | Alloc gate |
| T18 | Tier B worker pool saturation | High | P5-TIERB-OCCUPANCY-BUDGET | Occupancy budget (backlog) |
| T19 | Client RTT probe decoupled from tracker | High | P5-CLIENT-RTT-PROBE-CORRELATION | RTT correlation (backlog) |
| T20 | CH/Redis ingest burst / stream trim | High | P5-INGEST-SINK-BURST-RESILIENCE | Sink resilience (backlog) |
| T21 | Residential IPs in XDP blocklist (CGNAT collateral) | Critical | P5-XDP-RESIDENTIAL-POLICY-BOUNDARY | XDP vs FilterEngine split |
| T22 | Client-only safe-page reveal before server verdict | High | P5-HYBRID-SERVER-VERIFY-GATE | Server unlock path |
| T23 | Empty antifraud kinematics / HTTP replay | Medium | P5-ANTIFRAUD-SNAPSHOT-SCORING-GAPS | Kinematics scoring gaps |
| T24 | CGNAT `/24` / IP-reputation collateral | High | P3-CGNAT-L2-IP-COLLATERAL-BOUNDARY | `CGNAT_MOBILE_IP_BYPASS` (partial) |
| T25 | Apple Private Relay misclassified as DC | Medium | P3-APPLE-PRIVATE-RELAY-ASN-POLICY | Builtin AS13335/54113 + Apple UA exempt from DCASN |
| T26 | In-app WebView JA4 / Sec-CH false blocks | Medium | P3-INAPP-WEBVIEW-CLASSIFIER-HARDENING | `social_in_app` preset (partial) |
| T27 | Sync blocking I/O on click init (TTFB / white screen) | High | P2-CLICK-INGRESS-LATENCY-BUDGET | Filter deadline, proxy cap (partial) |
| T28 | Vision / multimodal moderation agents | High | P5-VISION-MULTIMODAL-MODERATION-AGENTS | Decoy parity; no server CV |
| T29 | CDP `Input.dispatch*` with `isTrusted: true` | High | P5-CDP-TRUSTED-INPUT-HARDENING | Kinematics heuristics; arms race |
| T30 | Custom Chromium + residential ASN | High | P5-CUSTOM-CHROMIUM-RESIDENTIAL-STACK | Cross-layer + crowd probe |

---

## T2: Human-in-the-loop Sybil (operator runbook)

**Threat:** Authorized or unauthorized human operators use real mobile devices on residential/cellular egress. Fingerprints are legitimate; automated threat intel feeds may not list the session. Attestation, crowd probe, and wave detection reduce risk but **do not replace** operational and contractual controls.

### What ingress cannot guarantee

- Block every third-party moderation agent or buyer-side auditor with a clean phone on LTE.
- Distinguish contractual authorized review from adversarial recon when device and network signals match organic users.
- Rely on client-only dwell time or pointer count thresholds alone (manual Sybil follows SOP).

### Required operational controls

| Control | Action |
| :--- | :--- |
| Scope | Separate **authorized audit** windows (known ASN / IP allowlist, credentials) from open production traffic. |
| Contracts | Moderator / network contracts define allowed surfaces; ingress flags implement policy, not legal scope. |
| Wave response | When `CrowdWaveActive` or admin wave API shows burst: **do not** widen production promotion; investigate before changing presets. |
| Intel hygiene | Do not bulk-promote residential/mobile IPs into XDP deny maps (see T21). |
| Buyer copy | Admin and sales docs must not promise "blocks all scrapers" or "guaranteed moderator block". |

### Signals to monitor

| Signal | Where |
| :--- | :--- |
| `ad_hybrid_crowd_wave_total{campaign_id}` | Prometheus |
| `GET /api/v1/fraud/crowd-waves/{campaign_id}` | Admin cold API |
| `crowd_probe_score`, `probe_cluster_id`, `crowd_wave_active` | ClickHouse click rows |
| `ad_crowd_probe_asn_signal_total` | Mobile ASN joint scoring (T12) |
| Safe-page verify rate vs click rate | Reports / CH drill |

### Response playbook

1. Confirm wave is not organic burst (geo, creative change, partner send).
2. If adversarial: keep `review_traffic_action=safe_page` or decoy; enable strict attestation + crowd wave gate.
3. Escalate to contractual channel for known moderator networks; do not depend on silent IP ban at `/24`.
4. Document incident in operator log; tune corpus only after FP review (T10).

### Verify

```bash
go test ./internal/filter/ -short -run CrowdWave -count=1
go test ./pkg/crowdwave/ -short -count=1
```

---

## T14: Client-edge DOM / on-device ML (residual)

Server ships DOM lint on lander publish and optional CSP on `/lp/`. **Third-party browser ML** (Chrome on-device inference, buyer lander scripts) runs outside tracker control. Mitigations: zone routing, decoy parity, CSP, honest operator docs. No server-side computer vision on `/track` hot path.

---

## T15: TLS server persona

Edge `:443` uses Mozilla Intermediate cipher profile in `deploy/nginx/snippets/ssl_server.conf`. Drift is gated in CI.

```bash
bash scripts/test/edge/tls_server_persona_audit.sh --config
bash scripts/test/edge/tls_server_persona_audit.sh --holdout
```

See `.cursor/rules/edge.mdc` **TLS server persona**.

---

## T21: XDP vs FilterEngine (residential policy)

| Layer | Use for residential / mobile |
| :--- | :--- |
| XDP | DC flood, token bucket, manual single-IP denies, syn subnet **rate** limit |
| Go `FilterEngine` | `ResidentialProxyFilter`, JA4 corpus, crowd probe, cross-layer |
| Forbidden | Auto-sync residential intel feed into BPF LPM at scale without CGNAT review |

---

## T24–T27: Conversion funnel false positives (ops audit)

| Defect | Mitigation slug | Notes |
| :--- | :--- | :--- |
| CGNAT IP collateral | P3-CGNAT-L2-IP-COLLATERAL-BOUNDARY | `ShouldBypassCGNATIPBlacklist` skips `blacklist:fraud` on mobile carrier when no `probe_cluster` corroboration; metric `ad_cgnat_collateral_skip_total` |
| Apple Private Relay | P3-APPLE-PRIVATE-RELAY-ASN-POLICY | `ApplePrivateRelayTable` exempts relay ASN + Apple UA from DCASN; metrics `ad_apple_relay_exempt_total` / `ad_apple_relay_reject_total`; refresh feed `netintel:apple_relay:v1` (cold worker backlog) |
| In-app WebView | P3-INAPP-WEBVIEW-CLASSIFIER-HARDENING | `InAppWebViewClassification` + `ad_inapp_webview_classified_total`; CH `in_app_webview_class`; enable `social_in_app` preset for social traffic |
| Click init latency | P2-CLICK-INGRESS-LATENCY-BUDGET | Budget table in `traffic.mdc`; `ad_click_ingress_latency_budget_exceeded_total{stage}`; no sync HTTP on `landing_bundle.go` |

### T24 operator drill

1. Confirm `CGNAT_MOBILE_IP_BYPASS=1` or campaign `cgnat_ip_policy_enabled` for mobile-heavy campaigns.
2. When a crawler poisons `blacklist:fraud` on carrier NAT, legit mobile clicks should not get L3 blocklist when bypass is on and probe cluster has not corroborated.
3. Verify: `go test ./internal/filter/ -short -run CGNAT -count=1`

### T25 operator drill

1. iOS Private Relay users (AS13335/54113 + iPhone/iPad UA) must not receive `datacenter_ip` from DCASN alone.
2. Windows UA on relay ASN still counts as datacenter (`ad_apple_relay_reject_total`).
3. Verify: `go test ./internal/filter/netintel/ -short -run PrivateRelay -count=1`

---

## T28–T30: Advanced adversary (honest limits)

| Vector | Mitigation | Residual |
| :--- | :--- | :--- |
| Vision / screenshot diff | Decoy parity, DOM lint, CSP; optional `content_diff_drill.sh` body hash | Public `/lp/{id}/` still scrapable; server does not run CV on hot path |
| CDP trusted synthetic input | Kinematics + verify scoring (`safe_page_behavior_test.go` holdouts) | Perfect CDP automation is arms race; `isTrusted` is not cryptographic proof |
| Custom Chromium + residential ASN | Cross-layer + crowd probe + probe cluster decoy route | All layers spoofed consistently may pass |

### T28 vision agents

Multimodal moderation pipelines scrape public landers out-of-band. Mitigations: zone routing, decoy parity, CSP. Optional operator drill:

```bash
CONTENT_DIFF_DRILL_URL=https://example/lp/1 bash scripts/test/edge/content_diff_drill.sh
bash scripts/test/edge/content_diff_drill.sh --holdout
```

### T29 CDP trusted input

Chrome DevTools `Input.dispatch*` can synthesize `isTrusted: true`. Safe-page verify uses kinematic curvature and timing variance; flat linear paths score below human corpus. Verify:

```bash
go test ./internal/track/ -short -run SafePageBehavior -count=1
```

### T30 custom Chromium residential stack

Residential ASN does not prove human operator. Joint scoring: cross-layer desync, `crowd_probe`, `probe_cluster` graph. When probe cluster routes, click decoys to safe view. Verify:

```bash
go test ./internal/filter/ -short -run 'CrossLayer|Residential|CrowdProbe' -count=1
```

---

## T12: Cross-session device reuse (mobile MVNO pools)

Coordinated reconnaissance after cookie or storage wipe is tracked via probe cluster graph (`P5-PROBE-CLUSTER-GRAPH`) and mobile ASN tier scoring (`P5-ASN-MOBILE-TIER`).

### Mobile ASN tier table (T0–T4)

| Tier | Meaning | Typical use |
| :--- | :--- | :--- |
| T0 | Unknown / non-mobile | No mobile-specific pessimism |
| T1 | Low-risk cellular | Monitor only |
| T2 | Regional carrier | Light joint scoring |
| T3 | MVNO / high-churn pool | `crowd_probe_asn` when probe score high |
| T4 | Known coordinated-scrape MVNO | Strongest joint scoring |

Builtin seed ASNs (override via feed file):

| ASN | Tier | Notes |
| :--- | :--- | :--- |
| 310410 | T4 | US T-Mobile |
| 58453 | T4 | China Mobile |
| 45400 | T4 | HK mobile |
| 26615 | T3 | US MVNO |
| 21928 | T3 | US MVNO |
| 20057 | T3 | US MVNO |
| 6167, 3215, 12479, 3209, 12956, 3320, 9808, 2856 | T2 | Major regional carriers |

Cold reload feed: `{MOBILE_ASN_TIER_FEED_DIR}/mobile_asn_tier.txt` (default `/var/lib/ad-event-processor/mobile-asn-tier/mobile_asn_tier.txt`).

Line format: `AS<number> <tier>` or `<number> <tier>` (tier 0–4). Comments start with `#`.

### Joint risk with ResidentialProxyRing and probe score

When `CROWD_PROBE_ENABLED=1` and `MOBILE_ASN_TIER_ENABLED=1`:

```
risk = alpha * ASN_Tier + beta * ProbeScore + gamma * ClusterHistory * 100
```

| Env | Default | Role |
| :--- | :--- | :--- |
| `CROWD_PROBE_ASN_WEIGHT_ALPHA` | 1.0 | ASN tier weight |
| `CROWD_PROBE_ASN_WEIGHT_BETA` | 0.5 | Probe behavior score weight |
| `CROWD_PROBE_ASN_WEIGHT_GAMMA` | 0.25 | Cluster / proxy ring history weight |

`ClusterHistory` is `1.0` when `ResidentialProxyRing` already shows a farm pattern for the campaign slot; otherwise Redis cluster prior millis / 1000.

### Fraud signal

`crowd_probe_asn` (L2 weak) fires when:

- `MobileTier(asn) >= CROWD_PROBE_ASN_MIN_TIER` (default 3)
- `ProbeBehaviorScore >= CROWD_PROBE_ASN_MIN_SCORE` (default 65)
- Campaign has safe page + attestation enabled (same gate as other crowd probe signals)

Metric: `ad_crowd_probe_asn_signal_total`.

### Verify

```bash
go test ./internal/filter/netintel/ -short -run Residential -count=1
go test ./internal/filter/ -short -run CrowdProbeFilter_holdoutASNMobileTier -count=1
```

---

## Operator checklist (pre-production and quarterly)

Use before enabling safe-page / attestation on high-spend campaigns and after major perimeter changes.

| Step | Check | Command / surface |
| :--- | :--- | :--- |
| 1 | Fraud limits doc reviewed with buyer | Admin: Documentation -> Fraud signal limits |
| 2 | Sybil / human operator runbook acknowledged | This doc, section T2 |
| 3 | Safe-page + attestation mode documented per campaign | Campaign editor fraud / advanced routing |
| 4 | Decoy parity drill | `bash scripts/test/edge/safe_page_parity_drill.sh` |
| 5 | TLS server persona audit | `bash scripts/test/edge/tls_server_persona_audit.sh --config` |
| 6 | XDP blocklist policy: no residential bulk sync | `P5-XDP-RESIDENTIAL-POLICY-BOUNDARY` |
| 7 | CGNAT policy env set for mobile-heavy traffic | `CGNAT_MOBILE_IP_BYPASS`, campaign `cgnat_ip_policy_enabled` |
| 8 | Crowd wave + probe cluster wired | `CROWD_WAVE_ENABLED`, `CROWD_PROBE_ENABLED` |
| 9 | No UI/doc claims "eliminated" Redis ops or guaranteed block of all agents | `bash scripts/ci/naming/antifraud_doc.sh` |
| 10 | Perimeter doc structure gate | `bash scripts/ci/naming/perimeter_intel_doc.sh` |

Artifact directory for TLS persona live probe (optional): `var/ci/tls_persona/`.

---

## Blue Team drills (reference)

| Drill | Script |
| :--- | :--- |
| Safe-page parity | `scripts/test/edge/safe_page_parity_drill.sh` |
| TLS persona | `scripts/test/edge/tls_server_persona_audit.sh` |
| First-party pixel | `scripts/test/edge/first_party_pixel_drill.sh` |
| Content diff (optional pHash) | `scripts/test/edge/content_diff_drill.sh` |

---

## Related

- `deploy/vendor/ANTIFRAUD.md`
- `docs/INTEGRATIONS.md` (safe-page / lander CSP)
- `ARBITRAGE_CLOSURE_BACKLOG.md` — P5 perimeter gap map
