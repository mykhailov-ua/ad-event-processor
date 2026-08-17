import { useCallback, useEffect, useState } from 'react';
import * as auth from '../helpers/auth.js';
import { can } from '../helpers/permissions.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { openRTBRoutingHint } from '../helpers/openrtb_endpoint.js';
import {
  fetchRtbIntegrationProfile,
  fetchRtbShadowDiff,
  validateBidRequest,
  applyRtbFloors,
  fetchRtbReconcileExport,
} from '../helpers/rtb_api.js';
import type { RtbFloorsApplyResult, RtbFloorSuggestionDTO, RtbReconcileExportDTO } from '../types/api/rtb.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { mapServiceError } from '../helpers/service_error.js';
import { formatAmountMicro } from '../helpers/money.js';
import { VALIDATE_BID_FIXTURE } from '../helpers/openrtb_endpoint.js';
import { to } from '../lib/to.js';
import { Button } from '../components/button.js';
import { ErrorBlock } from '../components/error_block.js';
import { SectionCard } from '../components/section_card.js';
import { StatusBadge } from '../components/status_badge.js';

const SMOKE_FIXTURE = VALIDATE_BID_FIXTURE;

type RtbRuntimeHints = {
  rtb_mode?: string;
  rtb_enabled?: boolean;
  rtb_exchange_no_bid_mode?: string;
};

type RtbEndpoints = {
  openrtb_bid_url?: string;
  edge_expose_openrtb?: boolean;
  edge_port_hint?: string;
  tracker_port_hint?: string;
};

type RtbIntegrationProfile = {
  openrtb_version?: string;
  required?: string[];
  supported?: string[];
  not_supported?: string[];
  runtime?: RtbRuntimeHints;
  endpoints?: RtbEndpoints;
};

type RtbShadowDiff = {
  source?: string;
  window?: string;
  shadow_evals?: number;
  parity_rate?: number;
  mismatch_rate?: number;
  shadow_no_bid?: number;
};

type ValidateBidResult = {
  valid?: boolean;
  errors?: string[];
};

function copyText(label: string, text: string) {
  navigator.clipboard?.writeText(text).then(() => {
    pushToastMessage({ title: 'Copied', message: `${label} copied` });
  }).catch(() => {
    pushToastMessage({ title: 'Copy failed', message: text });
  });
}

function CopyRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="integration-copy-row">
      <div className="integration-copy-row__head">
        <span className="form-label">{label}</span>
        <Button label="Copy" variant="secondary" size="sm" onClick={() => copyText(label, value)} />
      </div>
      <code className="code-block">{value}</code>
    </div>
  );
}

function BulletList({ items }: { items: string[] }) {
  return (
    <ul className="plain-list">
      {items.map((item) => (
        <li key={item} className="plain-list__item font-mono text-sm">{item}</li>
      ))}
    </ul>
  );
}

function RuntimeHints({ runtime }: { runtime?: RtbRuntimeHints | null }) {
  const mode = (runtime?.rtb_mode || 'off').toLowerCase();
  const enabled = runtime?.rtb_enabled === true;
  return (
    <dl className="definition-list">
      <dt>RTB_MODE</dt>
      <dd className="font-mono">{enabled ? mode : 'off (RTB disabled)'}</dd>
      <dt>RTB_EXCHANGE_NO_BID_MODE</dt>
      <dd className="font-mono">{runtime?.rtb_exchange_no_bid_mode || '—'}</dd>
      <dt>Note</dt>
      <dd className="text-muted text-sm">
        Env-driven on tracker/control. RTB_MODE=shadow|live affects in-auction path on /track;
        exchange partners use POST /openrtb/bid below.
      </dd>
    </dl>
  );
}

function ShadowSummary({ data }: { data: RtbShadowDiff | null }) {
  if (!data || data.source === 'unavailable') {
    return (
      <p className="text-muted text-sm">
        Shadow-diff metrics are recorded on tracker nodes. Control plane shows unavailable
        when not wired to hot-path counters.
      </p>
    );
  }
  const parity = data.parity_rate != null ? `${(data.parity_rate * 100).toFixed(1)}%` : '—';
  const mismatch = data.mismatch_rate != null ? `${(data.mismatch_rate * 100).toFixed(1)}%` : '—';
  return (
    <dl className="definition-list" data-testid="rtb-shadow-diff">
      <dt>Window</dt>
      <dd className="font-mono">{data.window ?? '1h'}</dd>
      <dt>Shadow evals</dt>
      <dd>{String(data.shadow_evals ?? 0)}</dd>
      <dt>Parity rate</dt>
      <dd>{parity}</dd>
      <dt>Mismatch rate</dt>
      <dd>{mismatch}</dd>
      <dt>Shadow no-bid</dt>
      <dd>{String(data.shadow_no_bid ?? 0)}</dd>
    </dl>
  );
}

/**
 * RTB exchange integration onboarding view.
 */
export function RtbIntegrationPage() {
  const user = auth.getUser();
  const canWrite = can(user?.permissions ?? [], 'rtb:write');
  const canApplyFloors = can(user?.permissions ?? [], 'settings:write');

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);
  const [profile, setProfile] = useState<RtbIntegrationProfile | null>(null);
  const [shadow, setShadow] = useState<RtbShadowDiff | null>(null);

  const [validateBusy, setValidateBusy] = useState(false);
  const [validateResult, setValidateResult] = useState<ValidateBidResult | null>(null);
  const [validateError, setValidateError] = useState<string | null>(null);
  const [fixtureText, setFixtureText] = useState(() => JSON.stringify(SMOKE_FIXTURE, null, 2));

  const [floorsBusy, setFloorsBusy] = useState(false);
  const [floorsResult, setFloorsResult] = useState<RtbFloorsApplyResult | null>(null);
  const [floorsError, setFloorsError] = useState<string | null>(null);

  const [reconcileBusy, setReconcileBusy] = useState(false);
  const [reconcileData, setReconcileData] = useState<RtbReconcileExportDTO | null>(null);
  const [reconcileError, setReconcileError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    const [profRes, shadowRes] = await Promise.all([
      to(fetchRtbIntegrationProfile()),
      to(fetchRtbShadowDiff('1h')),
    ]);
    if (profRes[1]) {
      setError(profRes[1]);
      setLoading(false);
      return;
    }
    setProfile((profRes[0] ?? null) as RtbIntegrationProfile | null);
    setShadow(shadowRes[1] ? null : (shadowRes[0] as RtbShadowDiff | null));
    setLoading(false);
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const runFloors = async (dryRun: boolean) => {
    if (!canApplyFloors || floorsBusy) return;
    setFloorsBusy(true);
    setFloorsError(null);
    if (dryRun) setFloorsResult(null);
    const [res, err] = await to(applyRtbFloors(dryRun));
    setFloorsBusy(false);
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      const view = mapServiceError(err);
      setFloorsError(view.message);
      pushToastMessage({ title: view.title, message: view.message, code: view.code });
      return;
    }
    const result = res as RtbFloorsApplyResult;
    setFloorsResult(result);
    if (!dryRun) {
      pushToastMessage({
        title: 'Floors applied',
        message: `${result.applied} deal(s), ${result.outbox_rows} outbox row(s)`,
      });
    }
  };

  const loadReconcileExport = async () => {
    if (reconcileBusy) return;
    setReconcileBusy(true);
    setReconcileError(null);
    const [res, err] = await to(fetchRtbReconcileExport('24h'));
    setReconcileBusy(false);
    if (err) {
      const view = mapServiceError(err);
      setReconcileError(view.message);
      pushToastMessage({ title: view.title, message: view.message, code: view.code });
      return;
    }
    const data = res as RtbReconcileExportDTO;
    setReconcileData(data);
    const json = JSON.stringify(data, null, 2);
    const blob = new Blob([json], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement('a');
    anchor.href = url;
    anchor.download = 'rtb-reconcile-export.json';
    anchor.click();
    URL.revokeObjectURL(url);
    pushToastMessage({
      title: 'Reconcile export',
      message: `${data.bids} bids, ${data.wins} wins`,
    });
  };

  const runValidate = async () => {
    setValidateBusy(true);
    setValidateError(null);
    setValidateResult(null);
    let body: unknown;
    try {
      body = JSON.parse(fixtureText);
    } catch (err) {
      setValidateBusy(false);
      setValidateError(`Invalid JSON: ${err instanceof Error ? err.message : String(err)}`);
      return;
    }
    const [res, err] = await to(validateBidRequest(body as Record<string, unknown>));
    setValidateBusy(false);
    if (err) {
      setValidateError(err instanceof Error ? err.message : String(err));
      return;
    }
    setValidateResult(res as ValidateBidResult | null);
  };

  if (error) {
    return <ErrorBlock error={error} />;
  }

  const endpoints = profile?.endpoints ?? {};
  const bidURL = endpoints.openrtb_bid_url ?? 'https://track.example/openrtb/bid';
  const routing = openRTBRoutingHint({
    edgeExposeOpenRTB: endpoints.edge_expose_openrtb,
    edgePortHint: endpoints.edge_port_hint,
    trackerPortHint: endpoints.tracker_port_hint,
  });
  const supported = profile?.supported ?? [];
  const suggestions = floorsResult?.suggestions ?? [];

  return (
    <section data-testid="rtb-integration-view">
      <div className="page-header">
        <h1 className="page-header__title">RTB integration</h1>
        <p className="page-header__desc">
          OpenRTB 2.6 exchange onboarding — profile, env hints, validate-bid smoke, shadow parity.
        </p>
        <p className="text-muted text-sm">
          <a href="/rtb/deals">← PMP deals</a>
        </p>
      </div>

      {loading ? <p className="loading-hint">Loading integration profile…</p> : null}

      {!loading && profile ? (
        <>
          <SectionCard
            title="Readiness checklist"
            icon="check-square"
            desc="Pre-flight gates before pointing exchange traffic at this stack."
          >
            <ul className="plain-list" data-testid="rtb-readiness-checklist">
              <li>{profile.openrtb_version ? '✓' : '○'} OpenRTB version declared</li>
              <li>{profile.runtime?.rtb_enabled ? '✓' : '○'} RTB enabled in runtime</li>
              <li>{endpoints.edge_expose_openrtb ? '✓' : '○'} Edge OpenRTB route exposed</li>
              <li>{validateResult?.valid ? '✓' : '○'} Validate-bid smoke passed</li>
              <li>{supported.length > 0 ? '✓' : '○'} Capability matrix loaded</li>
            </ul>
          </SectionCard>

          <SectionCard
            title="SSP endpoint"
            icon="globe"
            desc="Point exchange partners at this URL (chunked POST body allowed)."
          >
            <CopyRow label="POST /openrtb/bid" value={bidURL} />
            <p className="text-muted text-sm">{`Routing: ${routing}`}</p>
            {endpoints.edge_expose_openrtb ? (
              <StatusBadge status="ok" label="Edge OpenRTB enabled" />
            ) : (
              <StatusBadge status="neutral" label="Tracker ports only — enable edge in Platform settings" />
            )}
          </SectionCard>

          <SectionCard
            title="Runtime exchange settings"
            icon="settings"
            desc="Read-only env hints from this deployment (not editable in UI)."
          >
            <RuntimeHints runtime={profile.runtime} />
          </SectionCard>

          <SectionCard
            title="Integration profile"
            icon="clipboard-text"
            desc={`OpenRTB ${profile.openrtb_version ?? '2.6'} — required fields and capability matrix.`}
          >
            <h3 className="subsection-title">Required</h3>
            <BulletList items={(profile.required ?? []) as string[]} />
            <h3 className="subsection-title">Supported</h3>
            <BulletList items={supported.slice(0, 24)} />
            {supported.length > 24 ? (
              <p className="text-muted text-xs">{`+ ${supported.length - 24} more fields`}</p>
            ) : null}
            <h3 className="subsection-title">Not supported</h3>
            <BulletList items={(profile.not_supported ?? []) as string[]} />
          </SectionCard>

          <SectionCard
            title="Validate bid request"
            icon="check"
            desc="POST smoke test against control-plane validator (same rules as tracker exchange)."
          >
            <textarea
              className="form-input code-block"
              rows={14}
              value={fixtureText}
              onChange={(e) => setFixtureText(e.currentTarget.value)}
            />
            <div className="cluster--actions mt-2">
              <Button
                label={validateBusy ? 'Validating…' : 'Run validate-bid'}
                variant="primary"
                loading={validateBusy}
                disabled={validateBusy}
                onClick={() => void runValidate()}
              />
              <Button
                label="Reset fixture"
                variant="secondary"
                disabled={validateBusy}
                onClick={() => setFixtureText(JSON.stringify(SMOKE_FIXTURE, null, 2))}
              />
            </div>
            {validateError ? <p className="text-danger text-sm mt-2">{validateError}</p> : null}
            {validateResult ? (
              <div className="mt-2" data-testid="validate-bid-result">
                {validateResult.valid ? (
                  <StatusBadge status="ok" label="Valid OpenRTB 2.6 request" />
                ) : (
                  <StatusBadge status="error" label="Validation failed" />
                )}
                {validateResult.errors?.length ? (
                  <ul className="plain-list mt-2">
                    {validateResult.errors.map((msg) => (
                      <li key={msg} className="plain-list__item text-sm">{msg}</li>
                    ))}
                  </ul>
                ) : null}
              </div>
            ) : null}
          </SectionCard>

          <SectionCard
            title="Shadow diff (1h)"
            icon="git-compare"
            desc="Shadow vs live winner parity — gate before RTB_MODE=live."
          >
            <ShadowSummary data={shadow} />
          </SectionCard>

          <SectionCard
            title="Bid floor optimizer"
            icon="trending-up"
            desc="Preview ClickHouse-backed floor suggestions, then apply to deals (outbox propagation)."
          >
            {canApplyFloors ? (
              <div className="cluster--actions mb-3">
                <Button
                  label={floorsBusy ? 'Working…' : 'Preview'}
                  variant="secondary"
                  size="sm"
                  loading={floorsBusy}
                  disabled={floorsBusy}
                  data-testid="rtb-floors-preview"
                  onClick={() => void runFloors(true)}
                />
                <Button
                  label="Apply floors"
                  variant="danger"
                  size="sm"
                  loading={floorsBusy}
                  disabled={floorsBusy}
                  data-testid="rtb-floors-apply"
                  onClick={() => void runFloors(false)}
                />
              </div>
            ) : (
              <p className="text-muted text-sm">settings:write required to preview or apply floors.</p>
            )}
            {floorsError ? <p className="text-danger text-sm">{floorsError}</p> : null}
            {suggestions.length > 0 ? (
              <div className="table-wrapper">
                <table className="data-table">
                  <thead>
                    <tr>
                      <th scope="col">Placement</th>
                      <th scope="col">Deal</th>
                      <th scope="col">Current</th>
                      <th scope="col">Suggested</th>
                      <th scope="col">Win rate</th>
                    </tr>
                  </thead>
                  <tbody>
                    {suggestions.map((row: RtbFloorSuggestionDTO) => (
                      <tr key={`${row.placement_id}-${row.deal_id}`}>
                        <td className="font-mono text-xs">{row.placement_id}</td>
                        <td className="font-mono text-xs">{row.deal_id}</td>
                        <td className="font-mono">{formatAmountMicro(row.current_floor_micro)}</td>
                        <td className="font-mono">{formatAmountMicro(row.suggested_floor_micro)}</td>
                        <td>{`${(row.win_rate * 100).toFixed(1)}%`}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ) : floorsResult?.dry_run ? (
              <p className="text-muted text-sm">Preview returned no suggestions.</p>
            ) : null}
          </SectionCard>

          <SectionCard
            title="Reconcile export"
            icon="download"
            desc="Download ClickHouse bid/win snapshot and live-gate readiness for the last 24h."
          >
            <Button
              label={reconcileBusy ? 'Exporting…' : 'Download reconcile JSON'}
              variant="secondary"
              size="sm"
              loading={reconcileBusy}
              disabled={reconcileBusy}
              data-testid="rtb-reconcile-export"
              onClick={() => void loadReconcileExport()}
            />
            {reconcileError ? <p className="text-danger text-sm">{reconcileError}</p> : null}
            {reconcileData ? (
              <dl className="definition-list mt-2" data-testid="rtb-reconcile-summary">
                <dt>Bids</dt>
                <dd>{String(reconcileData.bids)}</dd>
                <dt>Wins</dt>
                <dd>{String(reconcileData.wins)}</dd>
                <dt>Spend (micro)</dt>
                <dd className="font-mono">{String(reconcileData.spend_micro)}</dd>
                <dt>Live gate</dt>
                <dd>{reconcileData.live_gate_ready ? 'Ready' : 'Not ready'}</dd>
              </dl>
            ) : null}
          </SectionCard>

          {!canWrite ? (
            <p className="text-muted text-sm">rtb:write required to create PMP deals.</p>
          ) : null}
        </>
      ) : null}
    </section>
  );
}
