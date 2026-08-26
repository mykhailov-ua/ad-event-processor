# Antifraud market analysis (internal)

Internal reference. Consolidates removed repo-root research notes (competitive landscape, practitioner critique, product extension proposal). Not shipped in customer packages.

Related: [ANTIFRAUD.md](./ANTIFRAUD.md) (implementation), [antifraud_backlog.md](./antifraud_backlog.md) (ROI ship list), [SALES.md](./SALES.md), [sku.yaml](./sku.yaml), [competitive_backlog.md](./competitive_backlog.md).

Historical sources (deleted from repo root, content retained here):

| Source | Original language | Role |
| :--- | :--- | :--- |
| Competitive antifraud survey (Russian) | Russian | Vendor landscape and feature blocks |
| Practitioner critique report (Russian PDF) | Russian | Forum pain points and vendor critique |
| Product extension proposal (Russian) | Russian | Four proposed features with internal realism notes in section 9 |

---

## 1. Industry context

Traffic arbitrage and CPA marketing operate in continuous conflict between traffic suppliers and automated invalid-traffic (IVT) generators. Botnets emulate behavioral signals, rotate residential and mobile proxies, and run antidetect browsers at scale. Classical IP blacklists and User-Agent checks are insufficient for modern fraud.

Buyers evaluate dedicated antifraud platforms (FraudScore, Anura.io, ClickCease), perimeter vendors (Cloudflare Bot Management), identification APIs (Fingerprint.com), and built-in tracker filters (Keitaro, Voluum).

Practitioner forums (Afflift, BlackHatWorld, STM Forum, Reddit r/adops, r/ppc, r/facebookads) report a gap between vendor marketing and production experience. Common complaints:

- opaque billing and limit accounting;
- missed headless and antidetect automation;
- false positives on legitimate mobile traffic;
- render-blocking client JS on landers;
- black-box risk scores without actionable evidence.

---

## 2. Competitive platform capabilities

Modern antifraud products differ by architecture and buyer segment. Six capability blocks drive subscription spend.

### 2.1 Smart conversion rejection (FraudScore SmartReject)

SmartReject automates settlement between ad networks, affiliates, and advertisers. Instead of post-hoc refund negotiations, it marks and blocks fraudulent conversions before they enter source reporting.

Scoring uses click-to-install histograms (TTI), in-app event interval distributions, and duplicate conversion detection. Each rejected conversion includes reason codes (VPN, datacenter proxy, geo mismatch) for payout disputes.

### 2.2 Deep hardware and behavioral identification (Fingerprint.com)

Fingerprint.com claims ~99.5% stable Visitor ID across IP changes, cookie clears, and incognito mode via Smart Signals.

Client-side signals include antidetect browser use, Canvas/WebGL spoofing, Android Emulator and iOS Simulator, root/jailbreak, Frida-style instrumentation, DevTools, synthetic touch input, and Puppeteer/Selenium automation.

### 2.3 Edge perimeter filtering (Cloudflare Bot Management)

Cloudflare filters IVT at the network edge before traffic reaches a tracker or lander. Each request gets a Bot Score from 1 (automation) to 99 (human).

Classification uses ML on global traffic, passive TLS fingerprints (JA3/JA4), HTTP/2 and HTTP/3 header ordering, and silent client JS checks. Verified Bots lists reduce accidental blocking of crawlers and monitors.

### 2.4 High-accuracy lead-gen positioning (Anura.io)

Anura targets high-cost-per-lead verticals (lead gen, finance, insurance) where false positives destroy margin. Marketing claims 99.999% detection via conservative thresholds.

Two modes: Anura Script (client JS, broad device coverage) and Anura Direct (REST API for pre-bid RTB and ping-post). Integrates with TrustedForm and LeadConduit for TCPA-related compliance workflows.

### 2.5 PPC campaign automation (ClickCease)

ClickCease (CHEQ Essentials) protects Google Ads, Microsoft Ads, and Meta Ads. Core workflow: API sync that adds malicious IPs and subnets to campaign exclusion lists in near real time.

Includes keyword-level attack monitoring for bid reductions or phrase shutdowns.

### 2.6 Built-in tracker filters (Keitaro, Voluum)

Keitaro runs filter rules locally on the arbitrageur server (low redirect latency). Rules cover geo, IP reputation, carriers, UA uniqueness, and client JS execution.

Voluum Anti-Fraud Kit scores traffic with Fast Click Ratio and click-to-conversion timing, assigning risk coefficients to traffic sources.

---

## 3. Competitive weaknesses and architectural limits

Production arbitrage traffic exposes structural limits in the products above.

### 3.1 False positives on 4G/5G CGNAT

IP reputation and frequency limits (ClickCease, baseline Cloudflare rules, default Keitaro filters) conflict with mobile carrier CGNAT. Thousands of real users share one public IPv4.

Filters see high request rates from a single IP and label it a botnet node. With Cloudflare Bot Management at medium sensitivity (score 2-29), managed challenge rates on legitimate mobile traffic can reach 15-20% in high-CGNAT regions (operator anecdote; threshold- and geo-dependent).

### 3.2 Antidetect browsers and residential proxies

Tracker built-ins and basic tracking scripts rely on datacenter IP lists and `navigator.webdriver` checks. Bot farms use AdsPower, Dolphin{anty}, Multilogin profiles with Chromium-level patches.

Traffic through clean residential or mobile proxies matches normal users for classical header and simple JS checks. Keitaro and Voluum entry rules pass this traffic when basic checks succeed.

### 3.3 Latency vs signal depth

| Product | Trade-off |
| :--- | :--- |
| Fingerprint.com | Client JS, fingerprint generation, `event_id` round-trip, backend `GET /v4/events/:event_id` adds ~150-400 ms per check; hurts pre-lander conversion |
| Anura Direct | Low-latency REST API but no client execution; loses WebGL, Canvas, and sensor signals |

### 3.4 Economics at high volume

| Vendor | Pricing shape | Arbitrage impact |
| :--- | :--- | :--- |
| Fingerprint Pro Plus | $99/mo for 20k API calls; $4 per additional 1k | ~$4k/mo at 1M identification calls erodes low-CPA margin |
| FraudScore | ~$390/mo entry | High barrier for solo buyers and small teams |
| Cloudflare Bot Management | Enterprise contracts | Not accessible to small self-serve buyers |

### 3.5 Post-factum analysis vs spend prevention

FraudScore provides deep conversion audit at postback time but does not block the paid click at CDN or DNS. The arbitrageur pays the ad network first, then disputes budget recovery.

---

## 4. Market pricing and billing failures

Fraud protection SaaS billing often creates cost independent of ad waste prevented.

### 4.1 Visit-based metering (ClickCease / CHEQ)

ClickCease tiers on total site traffic detected by its script, not paid PPC clicks alone. Limits consume organic, direct, repeat page views, and internal navigation.

Organic or content growth can force tier upgrades ($149-$349/mo) or overage charges without growth in paid traffic. CHEQ pricing defines traffic as any user activity the script detects or blocks, including internal page views.

Promotional pricing: advertised $69/mo applies for the first three billing cycles; regular Starter is ~$99/mo. Team features (permissions, white label) sit on Pro (~$149/mo). Annual plans bill upfront; monthly cancel-anytime costs more.

### 4.2 Enterprise entry pricing

Anura.io, 24metrics, and similar enterprise vendors use custom or usage-based contracts. Practitioner reports cite floors from ~$580/mo to $1,000+/mo or CPM-per-check models. Unsuitable for low-CPA popunder, push, and native funnels where 10-15% margin cannot absorb per-million-click SaaS fees.

### 4.3 Trial and overage mechanics

Forum and review reports describe difficult trial cancellation, accidental annual upgrades ($600-$1,700 non-refundable), and protection shutoff when monthly click caps are exceeded (Fraud Blocker).

### 4.4 Pricing comparison (practitioner view)

| Service | Headline price | Billing mechanics | Buyer pain |
| :--- | :--- | :--- | :--- |
| ClickCease (CHEQ) | $69-$99/mo base | All visits counted; promo 3 months; overage | Forced upgrade to $149-$349/mo without ad spend growth |
| Anura.io | Custom / enterprise | High minimum; usage-based | No fit for solo buyers under ~$5k/mo ad spend |
| ClickGuardian | From $49/mo | Ad spend bracket, not site visits | Highlights visit-metering downside of ClickCease |
| Fraud Blocker | $55-$69/mo | Protection off after cap | Mid-month budget exposure on traffic spikes |

---

## 5. Technology blind spots: bots and IP blocking

### 5.1 IP exclusion illusion (PPC)

ClickCease workflow: push suspicious IPs into Google Ads exclusion lists via API. Forum consensus: ineffective against professional fraud farms.

Google Ads caps IP exclusions at **500 per account/campaign**. Rotating mobile and residential pools overflow the list; older entries rotate out. A blocked bot changes IP and clicks again.

Practitioner rhetoric: IP-ban-centric tools miss the majority of real ad fraud (not a verified metric).

### 5.2 Headless browsers and fingerprint spoofing

Keitaro and Binom built-ins use datacenter lists, crawler signatures, and UA checks. Modern arbitrage bots run Headless Chrome, Puppeteer, Playwright with stealth plugins:

- spoof Canvas, WebGL, AudioContext, and font profiles;
- spline mouse paths and randomized click delays;
- pass `navigator.webdriver` and basic JS probes.

In lead gen and nutra, bots fill forms with junk leads to inflate CR inside low-quality networks while advertisers see zero lead quality.

---

## 6. False positives and clean mobile traffic

Aggressive fraud detection inflates "fraud found" metrics by blocking real users, raising CPA.

### 6.1 CGNAT collateral blocking

On 3G/4G/5G, one compromised device on a carrier gateway can cause ClickCease or Anura to blacklist the shared public IP. All clean users on that gateway are blocked.

Practitioner reports: strict ClickCease settings spiked CPA by cutting live mobile traffic classified as "repeat clicks from one IP."

### 6.2 VPN and residential privacy users

Anura and FraudScore heuristics score non-residential ASN patterns as high risk. This cuts paying Tier-1 users who routinely use VPN or residential privacy tools (US, UK, CA).

### 6.3 Cloudflare on arbitrage landers

Cloudflare as primary antifraud screen triggers HTTP 403 Suspected Phishing or endless Turnstile on clean arbitrage landers. Redirect-heavy traffic from specialty ad networks is often misclassified as phishing, killing conversion.

---

## 7. Lander latency and tracking degradation

Heavy client antifraud JS creates a paradox: protection reduces legitimate traffic performance.

### 7.1 Render blocking and bounce rate

Anura, CHEQ, and ClickCease behavioral analysis (mouse movement, WebGL GPU reads, DOM interaction) requires sync or async JS in page head. On mid-tier mobile devices this can block rendering for 1.5-3 seconds.

Arbitrage landers above ~1 second load time see sharp bounce increases. Users leave before antifraud and DOM complete; the buyer pays for a click that never reaches the offer.

### 7.2 Analytics and ad platform pixel conflict

Adops observations: GA4 undercounts 18-35% of paid traffic when:

- AdBlock/uBlock/Brave block ClickCease/Anura tags as trackers;
- users close tabs before heavy tags fire;
- ClickCease Pixel Guard conflicts with Meta and Google conversion pixels.

Missing conversion signals degrade Smart Bidding, Target CPA, and Target ROAS. Platforms under-optimize or shift auctions away from the intended audience.

---

## 8. Black-box scoring, refunds, and support

### 8.1 Black-box risk scores

Anura and FraudScore output aggregate risk ("Fraud Risk: 88%") without raw check logs. CPA network owners and media buyers cannot use these reports alone to claw back affiliate payouts or dispute traffic with ad networks. Counterparties reject scores without failed JS tests, packet anomalies, or reproducible technical evidence.

### 8.2 Ad network refund reality

Marketing claim: ClickCease helps recover Google Ads spend from invalid clicks. Practice: Google Invalid Clicks Dispute Form requires server logs with GCLID, timestamps, and IP. Practitioner reports: Google often rejects third-party ClickCease reports, stating internal IVT filtering already adjusted the auction and external criteria are invalid.

### 8.3 Support and FUD sales

Vendor reps participate in forums and subreddits emphasizing click fraud scale. Support for false-positive cases often recommends lowering filter sensitivity, which reduces protection without fixing misclassification.

| Factor | Field symptom | Impact |
| :--- | :--- | :--- |
| Black-box scoring | Risk percent without raw logs | Networks and affiliates reject fraud claims |
| Google/Meta refunds | Third-party reports declined | Subscription cost plus unrecovered click spend |
| Pixel distortion | Blocked or delayed JS tags | Smart Bidding and Target CPA degradation |
| FUD-driven sales | Fear-based forum marketing | Buyers adopt tools that underperform expectations |

---

## 9. Product extension proposal (four features)

Based on system architecture review and competitor weak points (FraudScore, Anura, FingerprintJS, ClickCease, Keitaro), four features were proposed to close arbitrage and CPA-network gaps and sharpen sales positioning.

**Status in tree (internal):** partial ship on four ROI backlog slugs (`antifraud_backlog.md`): conversion smart reject, automation fraud metrics, canvas test-retest, CGNAT IP velocity bypass. Still not shipped: full mobile biometrics pipeline, signed evidence-pack export. CGNAT v1 skips `ipv4_rotation` and ingress RPD only for tier-1 MNO ASNs when `cgnat_ip_policy_enabled` or `CGNAT_MOBILE_IP_BYPASS` (`ANTIFRAUD.md`); does not disable IP reputation globally or add gyro/touch scoring.

### 9.1 Antidetect noise injection probing

**Market problem:** Antidetect browsers (AdsPower, Dolphin{anty}, Multilogin, Octo Browser) patch Chromium at native level and inject pseudo-random micro-noise into Canvas and WebGL fingerprints. Basic tracker filters treat noisy fingerprints as unique humans.

**Proposed implementation:**

- Lightweight client script rendering a hidden Canvas/WebGL scene.
- Analyze floating-point rounding artifacts during graphics rendering; antidetect noise generators leave programmatic signatures absent on real GPUs and mobile SoCs.
- Flag sessions as antidetect bot without relying on IP, including behind clean mobile or residential proxies.

**Sales angle:** Keitaro and Voluum pass AdsPower/Dolphin traffic; positioning as hardware-level noise detection rather than IP reputation.

**Internal realism notes:**

- Test-retest Canvas coherence is an active industry technique (Castle.io, Group-IB, Sentinel); it is not a solved one-shot feature.
- Claimed 99.8% accuracy in the source proposal is marketing-grade without benchmark protocol.
- Tracker hot-path SLA (p95 < 50 ms on `/track`) conflicts with heavy client probing on redirect-only chains.
- Antidetect vendors continuously adapt noise algorithms.

### 9.2 Mobile session disambiguation (4G/5G CGNAT)

**Market problem:** CGNAT causes IP-frequency tools to block legitimate mobile users; practitioners cite 15-20% clean mobile loss.

**Proposed implementation:**

- When traffic is classified as mobile carrier, disable IP frequency limits and IP reputation scoring.
- Shift scoring to passive device biometrics: accelerometer/gyroscope micro-motion, scroll tilt changes, touch radius, inter-touch and inter-scroll interval entropy.
- Accept high volume from one cellular IP when physical interaction profiles differ; block when input is absent or perfectly periodic.

**Shipped v1 (velocity only):** campaign `cgnat_ip_policy_enabled` + optional `CGNAT_MOBILE_IP_BYPASS` skip `ipv4_rotation` and ingress RPD for builtin tier-1 MNO ASN set (`mobile_carrier_asn_table.go`). `datacenter_ip`, TLS, L3 blacklist, and Lua budget unchanged. Biometrics and global IP reputation disable remain backlog.

**Sales angle:** Do not cut 4G/5G CGNAT traffic; distinguish live finger from emulator.

**Internal realism notes:**

- CGNAT pain is real; blanket IP ignore for all cellular traffic is risky because mobile proxies are standard bot-farm tooling.
- Biometrics require client JS on lander or safe page; redirect-only tracker paths see no sensor data.
- iOS/Safari and privacy browsers restrict or noise sensor APIs.

### 9.3 Fraud evidence pack (automated export)

**Market problem:** Antifraud tools report aggregate fraud rates. When arbitrageurs dispute spend with ad networks (ExoClick, PropellerAds, TrafficJunky), networks demand hard evidence; without it refunds are denied.

**Proposed implementation:**

- Admin button: "Generate Fraud Evidence Pack" for a campaign or time range.
- Auto-assembled PDF and JSON with timestamps, compromised publisher/zone IDs, network anomaly dumps, fingerprint mismatch hashes, Canvas/WebGL emulation indicators.
- Platform digital signature attesting report authenticity.
- Export ready for ad network support tickets.

**Sales angle:** Position as refund tooling, not only software cost.

**Internal realism notes:**

- High value for **internal** disputes (affiliate clawback, CPA network payout cuts).
- Google and major PPC platforms rarely accept third-party PDFs as invalid-click proof (see section 8.2).
- Not implemented in control plane or export pipeline today.

### 9.4 Modular feature flags in admin UI

**Market problem:** Fixed full-stack subscription creates "why pay for everything" objection.

**Proposed implementation:**

- Admin toggles at subscription setup: base edge filter and cloaking; antidetect noise module; 4G/5G biometrics module; OpenRTB integration.
- Backend maps routes and filter engine capabilities to license flags.

**Sales angle:** Entry at lower monthly check with upsell to full stack.

**Internal alignment:** JWT feature gating already exists (`ivt_ml_detector`, `ml_fraud_boost`, `openrtb_engine`, etc. in `sku.yaml`). Current tiers: Starter $129, Pro $329, Scale $649, Network $1,199, Enterprise $2,500+ (not the $150/$300 figures in the source proposal). UI module toggles per antidetect/biometrics slice are not shipped.

---

## 10. Conclusions and market direction

### 10.1 Practitioner critique summary

Boxed antifraud products, especially IP-block-centric PPC tools, often fail operational expectations:

- subscription and overage cost exceeds saved waste on low-margin funnels;
- false positives on 3G/4G/5G CGNAT;
- lander slowdown from render-blocking JS;
- missed headless and antidetect automation at scale.

Community direction (forum consensus in source reports):

- server-side telemetry validation;
- cleaner upstream traffic sources;
- deep postback analysis;
- custom filters inside trackers without heavy third-party JS.

### 10.2 ad-event-processor positioning (internal)

| Axis | Typical SaaS competitor | This product |
| :--- | :--- | :--- |
| Model | SaaS per visit or per check | Self-hosted on-prem; RPS and host limits |
| Entry price | $49-$99/mo SaaS | $129/mo Starter (`sku.yaml`) |
| PPC Google sync | ClickCease, ClickGuardian | Not in current scope |
| Edge signals | Vendor-managed | TLS/JA3, TCP MSS when edge forwards headers; CDN limits documented in `ANTIFRAUD.md` |
| Client fingerprint | Fingerprint, Anura Script | Safe-page attestation (basic WebGL/Canvas) |
| ML / IVT | Vendor-operated | Cold-path `ivt-detector`, `fraud-scorer`; buyer-operated |
| Antidetect noise / CGNAT / evidence pack | Partial at enterprise vendors | Proposed; not shipped |

Strong fit: redirect and S2S arbitrage, self-hosted CPA networks, buyers who want appliance control and license JWT entitlements.

Weak fit without further work: PPC click-fraud workflow (Google/Meta API exclusion sync), guaranteed ad-network refunds via exported PDFs, marketing accuracy claims without holdout benchmarks.

### 10.3 Claims discipline for vendor docs

Do not export to customer-facing copy without evidence:

- fixed accuracy percentages (99.8%, 99.999%);
- "legal proof" for ad network refunds;
- full antidetect detection on redirect-only paths without lander JS;
- CGNAT handling via complete IP ignore for cellular traffic.

Canonical implementation truth: [ANTIFRAUD.md](./ANTIFRAUD.md), [competitive_backlog.md](./competitive_backlog.md).

---

## Appendix: vendor capability quick reference

| Capability | FraudScore | Fingerprint | Cloudflare | Anura | ClickCease | Keitaro/Voluum |
| :--- | :---: | :---: | :---: | :---: | :---: | :---: |
| Real-time click block at edge | Limited | Via integration | Yes | Script/Direct | PPC exclusions | Redirect rules |
| Conversion SmartReject | Yes | No | No | Yes | No | Limited |
| Client Canvas/WebGL depth | Partial | Yes | Partial | Yes | Partial | Basic |
| CGNAT-safe mobile policy | Weak | Weak | Weak | Weak | Weak | Weak |
| Low-latency redirect path | N/A | Poor (API chain) | Good | Direct API only | JS on lander | Good (local) |
| Self-hosted / no per-check SaaS | No | No | No | No | No | Yes (tracker only) |
| Actionable evidence export | Partial | API logs | Logs | Partial | Reports | Logs |

Pricing references (verify at sale time): Fingerprint Pro Plus $99/20k + $4/1k ([fingerprint.com/pricing](https://fingerprint.com/pricing)); ClickCease visit-metered tiers on [CHEQ pricing](https://essentials.cheq.ai/pricing); Anura custom usage-based ([anura.io](https://www.anura.io/product)).
