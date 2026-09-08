# Perimeter & commercial intelligence protection (AppSec)

Blue Team reference: unauthorized third-party scanners, distributed coordination attacks (Sybil), and commercial intelligence scraping against **Public Safe Sandbox Zone** vs **Internal Trusted Production Zone** separation.

Cross-ref: `ARBITRAGE_CLOSURE_BACKLOG.md` (P5 slugs), `deploy/vendor/ANTIFRAUD.md`, `internal/track/safe_page_attest.go`.

Code symbols (`safe_page_*`, `review_traffic_*`) unchanged; this document uses zone names only.

---

## Threat actors

| Actor | Capability | Objective |
| :--- | :--- | :--- |
| Automated scanner fleet | Datacenter + consumer IP, headless Chrome | Dual-origin differential scrape |
| Unauthorized third-party agent | Real mobile, cellular ASN, `isTrusted` events | API surface mapping, IP theft |
| Coordinated Sybil pool | Tier-3 human operators + SOP scripts | Production zone reconnaissance |
| Commercial intelligence vendor | TLS/JA4/WebGL corpus | Fingerprint production ingress policy |

---

## Threat catalog

| ID | Threat | Ingress signal | Residual risk |
| :--- | :--- | :--- | :--- |
| T1 | Dual-origin fetch: sandbox HTML/URL ≠ production | Paired egress diff | Medium |
| T2 | Human-in-the-loop Sybil (residential cellular) | No intel-feed match | High (governance) |
| T3 | Empty sandbox hydrator | Attestation API exposed | High until P5 hydrator |
| T4 | Static sandbox DOM ≠ production asset graph | Script/DOM hash delta | High |
| T5 | Multi-hop redirect (DMR) | Navigation timing signature | Medium |
| T6 | Third-party telemetry script | Cross-site script classification | Medium |
| T7 | Server event without browser correlate | Postback integrity gap | Medium |
| T8 | iframe sandbox embedding | DOM nesting probe | Medium |
| T9 | Internal zone response headers | Infra fingerprint | Low (edge strips) |
| T10 | Threat intel corpus false positive | Legit user to sandbox | Medium |
| T11 | Coordinated probe micro-behavior | SOP scroll/click timing | Medium |
| T12 | Cross-session device reuse | Cookie/storage wipe | Medium |
| T13 | Hybrid AI + crowd waves | Distributed testing bursts | Medium |
| T14 | Client-edge DOM ML | Buyer browser inference | High |
| T15 | TLS server persona scan | L4 infra fingerprint | Medium |
| T16 | Routing timing side-channel | TTFB differential | Medium |

---

## Zone model

| Zone | Code path | Purpose |
| :--- | :--- | :--- |
| Public Safe Sandbox Zone | `safe_view.go`, `safe_page_stub`, `review_traffic_policy` route | Untrusted ingress; limited asset exposure |
| Internal Trusted Production Zone | Accepted click redirect, production lander, debit path | Authorized user business flow |

Separation controls: `review_traffic_policy`, threat intel corpus (`pkg/moderatorcorpus`), attestation (`POST /track/verify`), cross-layer desync policy.

---

## Per-threat remediation

### T1 — Dual-origin scrape

**Detection:** Two egress classes fetch same ingress URL; compare `sha256(body)`, redirect depth, script `src` list.

**Controls:** P5-DECOY-LANDING-PARITY, P5-CONTENT-DIFF-DRILL, threat intel corpus routing.

**Verify:** `bash scripts/test/edge/safe_page_parity_drill.sh`

### T2 — Sybil human operator

**Detection:** Legitimate device fingerprint; no automated intel match.

**Controls:** Operational governance only. P5-SYBIL-HUMAN-OPERATOR-RUNBOOK.

### T3 — Empty hydrator

**Detection:** Sandbox stub without client attestation probe.

**Controls:** P5-SAFE-PAGE-HYDRATOR-CLIENT implements full `SafePageVerifyFingerprint` POST.

### T4 — DOM differential

**Detection:** Sandbox template graph ≠ production lander shell.

**Controls:** Hosted decoy template parity; P5-BEHAVIOR-MODEL-HUMANIZATION.

### T5 — Redirect chain

**Controls:** P5-REDIRECT-PROFILE-COMPLIANCE (`302` strict default).

### T6 — Third-party script

**Controls:** P5-FIRST-PARTY-PIXEL-ORIGIN same-origin `/_aed/track.js`.

### T7 — Event integrity

**Controls:** P5-CAPI-BROWSER-DEDUP; shared `event_id`; suppress postback on sandbox-routed ingress.

### T11 — Coordinated probe behavior

**Features:** path efficiency, scroll CV, Fitts residual, event-order entropy, footer-reach ms, `isTrusted` ratio.

**Controls:** P5-CROWD-PROBE-SCORING, signals `crowd_probe_behavior`, `crowd_probe_timing`.

### T12 — Cross-session clustering

**Cluster key:** HMAC(canvas, audio, webgl, tls ja3/ja4, tcp sig, screen, font hash).

**Controls:** P5-PROBE-CLUSTER-GRAPH (Redis), P5-ASN-MOBILE-TIER.

---

## Client JS surface

| File | Zone | Gap |
| :--- | :--- | :--- |
| `web/src/static/track.js` | Production telemetry | Third-party origin; no dwell gate |
| `internal/track/track_telemetry.js` | Both | Integer coords; no `isTrusted` |
| `internal/track/safe_page_hydrator.js` | Sandbox | **Empty** |

---

## Operator checklist

1. Run P5-CONTENT-DIFF-DRILL before production traffic scale-up.
2. Confirm threat intel corpus and `review_traffic_action` for traffic source.
3. Align browser and server `event_id` when both paths active.
4. Ship hydrator before strict attestation mode.
5. Document T2 limits in admin fraud panel.
6. CDN ingress: disable OS fingerprint flags per `ANTIFRAUD.md`.

---

## P5 backlog index

| Slug | Threats |
| :--- | :--- |
| P5-SAFE-PAGE-HYDRATOR-CLIENT | T3, T8 |
| P5-DECOY-LANDING-PARITY | T1, T4 |
| P5-CONTENT-DIFF-DRILL | T1 |
| P5-CAPI-BROWSER-DEDUP | T7 |
| P5-REDIRECT-PROFILE-COMPLIANCE | T5 |
| P5-FIRST-PARTY-PIXEL-ORIGIN | T6 |
| P5-BEHAVIOR-MODEL-HUMANIZATION | T4, T11 |
| P5-CROWD-PROBE-SCORING | T11 |
| P5-PROBE-CLUSTER-GRAPH | T12 |
| P5-ASN-MOBILE-TIER | T12 |
| P5-SYBIL-HUMAN-OPERATOR-RUNBOOK | T2 |
| P5-CLICK-TIMING-WIRE | T11 |

---

## 2026 threat extensions (T13–T16)

Attack surface shifts from datacenter scanner IPs to **hybrid human chains** and **buyer-browser client-edge inference**. Server-only controls are insufficient for T14.

### T13 — Hybrid AI + crowd wave reconnaissance

**Mechanism:** Upstream platform AI flags Public Safe Sandbox Zone (no production telemetry correlate). System queues distributed human operators (crowd compute marketplaces) with clean cellular TLS/JA4/`isTrusted` events. Same ingress URL attacked in **waves** of distinct devices to infer zone transition rules via behavioral differential.

**Current gap:** `review_traffic_policy` matches per-request intel; no **campaign-level wave** detector. Attestation promotion may succeed for individual probe sessions.

**Detection:**

| Signal | Rule |
| :--- | :--- |
| `uniq(cluster_id)` per campaign per hour | > τ_wave_devices |
| Behavior simhash Hamming neighborhood | ≥ M templates within W seconds |
| `verify_post / click` ratio | Spike without conversion correlate |
| Geographic ASN diversity + low organic CTR | Joint with `ResidentialProxyRing` |

**Controls:** P5-HYBRID-CROWD-WAVE-DETECTION — block production zone promotion during active wave score; feed threat intel corpus.

**Residual:** Individual operator with single session still maps to T2.

---

### T14 — Client-edge DOM analysis (on-device ML)

**Mechanism:** Inference runtime inside buyer Chrome analyzes Internal Trusted Production Zone DOM (link graph, hidden redirects, overlay patterns). Signatures egress from **client memory** to third-party backends. Ingress SSR, `review_traffic_policy`, and tracker filters are **not on request path**.

**Current gap:** No lander publish lint; production `/lp/` assets operator-controlled; DMR/meta patterns may exist on production zone.

**Controls:**

| Layer | Action |
| :--- | :--- |
| Publish gate | P5-CLIENT-EDGE-DOM-INTEGRITY lint before `/lp/` deploy |
| CSP | `Content-Security-Policy` on static lander host |
| Architecture | No server-only redirects visible only after routing split |
| Monitoring | External account health only; no server metric for on-device ML |

**Residual:** High. Client-edge classifiers cannot be disabled from ad-event-processor.

---

### T15 — TLS server persona interrogation (L4)

**Mechanism:** Scanner opens TLS to edge/tracker; classifies **server** cipher suite order, extensions, ALPN, HTTP/2 SETTINGS — independent of HTML. Non-interoperable or «hardened anomaly» profile tagged as protection intermediary → infra-level action.

**Current gap:** No checked baseline for nginx `ssl_ciphers` / http2 directives; P4 H2 ingress trace not yet shipped.

**Controls:** P5-TLS-SERVER-PERSONA-HARDENING — Mozilla Intermediate-aligned profile, CI audit script, drift gate.

**Note:** Client JA3/JA4 on `fetch` (T6) is separate from **server** persona (T15).

---

### T16 — Routing timing side-channel

**Mechanism:** Scanner holds session open, delays JS telemetry, measures `GET /click` TTFB and connection lifecycle. If sandbox route returns faster than production route (early `review_traffic` vs full filter chain), external observer infers hidden routing.

**Current gap:** `review_traffic_policy` short-circuits before `FilterEngine`; different response sizes (safe view vs 302); no timing pad.

**Controls:**

| Item | Action |
| :--- | :--- |
| Timing pad | P5-ROUTING-TIMING-CONSTANT-TIME minimum response bucket |
| Body size buckets | Pad sandbox HTML to production response size class where feasible |
| Async verify | Never block initial click TCP on verify POST wait |
| Drill | Extend `safe_page_parity_drill.sh` with TTFB delta assertion |

**Internal use:** P5-CLICK-TIMING-WIRE populates CH for **defender** analytics; must not leak route class externally.

---

## P5 backlog index (extended)

| Slug | Threats |
| :--- | :--- |
| P5-HYBRID-CROWD-WAVE-DETECTION | T13 |
| P5-CLIENT-EDGE-DOM-INTEGRITY | T14 |
| P5-TLS-SERVER-PERSONA-HARDENING | T15 |
| P5-ROUTING-TIMING-CONSTANT-TIME | T16 |
