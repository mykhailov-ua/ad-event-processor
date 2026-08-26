import { useCallback, useEffect, useState, type ReactNode } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import type { DataFreshness } from '../types/report.js';
import { to } from '../lib/to.js';
import * as auth from '../helpers/auth.js';
import { hasBoundCustomer, boundCustomerId } from '../helpers/buyer_session.js';
import { validateReportRange } from '../helpers/validators.js';
import { REPORT_DATE_PRESETS } from '../helpers/date_presets.js';
import { tenantReportQueryString } from '../helpers/tenant_url.js';
import { formatMoney } from '../helpers/money.js';
import {
  fetchClickLog,
  type ClickLogEvent,
  type ClickLogPostback,
} from '../helpers/click_log_api.js';
import { AlertBanner } from '../components/alert_banner.js';
import { Button } from '../components/button.js';
import { ErrorBlock } from '../components/error_block.js';
import { FormField } from '../components/form_field.js';
import { FreshnessBadge } from '../components/freshness_badge.js';

const PAGE_PATH = '/reports/clicks';

/**
 * Click log drill-down: search by click_id and view click/conversion timeline.
 */
export function ClickLogReportPage() {
  const [searchParams] = useSearchParams();
  const user = auth.getUser();
  const sessionScoped = hasBoundCustomer(user?.role);
  const preset = REPORT_DATE_PRESETS[1] ?? REPORT_DATE_PRESETS[0];

  const [customerInput, setCustomerInput] = useState(
    searchParams.get('customer_id') || (sessionScoped ? boundCustomerId(user) : '')
  );
  const [from, setFrom] = useState(searchParams.get('from') || preset.from());
  const [rangeTo, setRangeTo] = useState(searchParams.get('to') || preset.to());
  const [clickId, setClickId] = useState(searchParams.get('click_id') || '');
  const [campaignId, setCampaignId] = useState(searchParams.get('campaign_id') || '');
  const [loading, setLoading] = useState(false);
  const [events, setEvents] = useState<ClickLogEvent[]>([]);
  const [postbacks, setPostbacks] = useState<ClickLogPostback[]>([]);
  const [freshness, setFreshness] = useState<DataFreshness | null>(null);
  const [error, setError] = useState<unknown>(null);
  const [validationError, setValidationError] = useState<string | null>(null);
  const [searched, setSearched] = useState(false);

  const runSearch = useCallback(
    async (overrides?: { clickId?: string; campaignId?: string }) => {
      const cid = sessionScoped ? boundCustomerId(user) : customerInput.trim();
      const rangeErr = validateReportRange(from, rangeTo);
      const trimmedClick = (overrides?.clickId ?? clickId).trim();
      const trimmedCampaign = (overrides?.campaignId ?? campaignId).trim();
      if (!cid) {
        setValidationError(null);
        setEvents([]);
        setPostbacks([]);
        setError(null);
        return;
      }
      if (rangeErr) {
        setValidationError(rangeErr);
        setEvents([]);
        setPostbacks([]);
        setError(null);
        return;
      }
      if (!trimmedClick && !trimmedCampaign) {
        setValidationError('Enter click_id or campaign_id');
        setEvents([]);
        setPostbacks([]);
        setError(null);
        return;
      }
      setValidationError(null);
      setLoading(true);
      setError(null);
      const [res, err] = await to(
        fetchClickLog({
          customerId: cid,
          from,
          to: rangeTo,
          clickId: trimmedClick || undefined,
          campaignId: trimmedCampaign || undefined,
        })
      );
      setLoading(false);
      setSearched(true);
      if (err) {
        setError(err);
        return;
      }
      if (overrides?.clickId != null) {
        setClickId(overrides.clickId);
      }
      setEvents(res?.events ?? []);
      setPostbacks(res?.postbacks ?? []);
      setFreshness(res?.freshness ?? null);
      const qsParts: Record<string, string> = { customer_id: cid, from, to: rangeTo };
      if (trimmedClick) qsParts.click_id = trimmedClick;
      if (trimmedCampaign) qsParts.campaign_id = trimmedCampaign;
      window.history.replaceState(null, '', `${PAGE_PATH}?${tenantReportQueryString(qsParts)}`);
    },
    [sessionScoped, user, customerInput, from, rangeTo, clickId, campaignId]
  );

  const load = useCallback(() => runSearch(), [runSearch]);

  useEffect(() => {
    const hasSeed =
      (searchParams.get('click_id') || searchParams.get('campaign_id')) &&
      (sessionScoped || searchParams.get('customer_id'));
    if (hasSeed) {
      void load();
    }
  }, []);

  if (error) return <ErrorBlock error={error} />;

  const timelineMode = clickId.trim() !== '';

  return (
    <>
      <div className="page-header">
        <h1 className="page-header__title">Click log</h1>
        <p className="text-muted text-sm">
          <Link to="/reports">{'<-'} Reports hub</Link>
        </p>
        {freshness ? (
          <FreshnessBadge stale={freshness.stale} lagSeconds={freshness.ch_lag_seconds} />
        ) : null}
      </div>

      <form
        className="mb-4"
        onSubmit={(e) => {
          e.preventDefault();
          void load();
        }}
      >
        {!sessionScoped ? (
          <FormField label="Customer ID">
            <input
              className="input"
              value={customerInput}
              onChange={(e) => setCustomerInput(e.target.value)}
              placeholder="customer uuid"
            />
          </FormField>
        ) : null}
        <div className="form-row">
          <FormField label="From">
            <input
              className="input"
              type="date"
              value={from}
              onChange={(e) => setFrom(e.target.value)}
            />
          </FormField>
          <FormField label="To">
            <input
              className="input"
              type="date"
              value={rangeTo}
              onChange={(e) => setRangeTo(e.target.value)}
            />
          </FormField>
        </div>
        <div className="form-row">
          <FormField label="Click ID">
            <input
              className="input font-mono"
              value={clickId}
              onChange={(e) => setClickId(e.target.value)}
              placeholder="click uuid"
            />
          </FormField>
          <FormField label="Campaign ID (browse)">
            <input
              className="input font-mono"
              value={campaignId}
              onChange={(e) => setCampaignId(e.target.value)}
              placeholder="optional campaign uuid"
            />
          </FormField>
        </div>
        {validationError ? <AlertBanner variant="warning" message={validationError} /> : null}
        <Button
          label={loading ? 'Loading...' : 'Search'}
          type="submit"
          disabled={loading}
          loading={loading}
        />
      </form>

      {searched && events.length === 0 ? (
        <p className="text-muted">No events in range for the given filters.</p>
      ) : null}

      {timelineMode && postbacks.length > 0 ? (
        <Section title="Outbound postbacks">
          <table className="data-table data-table--compact">
            <thead>
              <tr>
                <th>Time</th>
                <th>Status</th>
                <th>Error</th>
              </tr>
            </thead>
            <tbody>
              {postbacks.map((row, idx) => (
                <tr key={`${row.created_at}-${idx}`}>
                  <td>{row.created_at}</td>
                  <td>{row.status}</td>
                  <td>{row.error_message || '-'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </Section>
      ) : null}

      {events.length > 0 ? (
        <Section title={timelineMode ? 'Event timeline' : 'Recent clicks'}>
          <table className="data-table data-table--compact" data-testid="click-log-table">
            <thead>
              <tr>
                <th>Time</th>
                <th>Type</th>
                <th>Click ID</th>
                <th>Campaign</th>
                <th>Placement</th>
                <th>Cost</th>
                <th>Revenue</th>
                <th>Status</th>
                <th>Sub1</th>
              </tr>
            </thead>
            <tbody>
              {events.map((row, idx) => (
                <tr key={`${row.created_at}-${row.event_type}-${idx}`}>
                  <td>{row.created_at}</td>
                  <td>{row.event_type}</td>
                  <td>
                    {!timelineMode ? (
                      <button
                        type="button"
                        className="link-button font-mono"
                        onClick={() => {
                          void runSearch({ clickId: row.click_id });
                        }}
                      >
                        {row.click_id}
                      </button>
                    ) : (
                      <span className="font-mono">{row.click_id}</span>
                    )}
                  </td>
                  <td>
                    <Link to={`/campaigns/${row.campaign_id}`}>{row.campaign_id}</Link>
                  </td>
                  <td>{row.placement_id || '-'}</td>
                  <td>
                    {row.attributed_cost_micro ? formatMoney(row.attributed_cost_micro) : '-'}
                  </td>
                  <td>{row.revenue_micro ? formatMoney(row.revenue_micro) : '-'}</td>
                  <td>{row.inbound_status || row.goal_name || '-'}</td>
                  <td>{row.sub1 || '-'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </Section>
      ) : null}
    </>
  );
}

/**
 * Section wrapper for click log tables.
 */
function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="mb-6">
      <h2 className="text-lg font-semibold mb-2">{title}</h2>
      {children}
    </section>
  );
}
