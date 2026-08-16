import type { ReactNode } from 'react';
import { TELEGRAM_REPORT_PAGES } from '../../helpers/nav_reports.js';
import { REPORT_DATE_PRESETS } from '../../helpers/date_presets.js';
import { telegramPiiNotice } from '../../helpers/tg_pii.js';
import { buyerEmptyCopy } from '../../models/empty_state.js';
import { telegramReportHref } from '../../helpers/telegram_report_state.js';
import { Button } from '../../components/button.js';
import { ButtonLink } from '../../components/button.js';
import { FormField } from '../../components/form_field.js';
import { FreshnessBadge } from '../../components/freshness_badge.js';
import type { useTelegramReport } from '../../hooks/use_telegram_report.js';

type TelegramReportShellProps = {
  title: string;
  pagePath: string;
  freshness?: { stale?: boolean; lag_seconds?: number; ch_lag_seconds?: number } | null;
  loading: boolean;
  tg: ReturnType<typeof useTelegramReport>;
  onSubmit: () => void;
  children: ReactNode;
  showExport?: boolean;
};

export function TelegramReportShell({
  title,
  pagePath,
  freshness,
  loading,
  tg,
  onSubmit,
  children,
  showExport = true,
}: TelegramReportShellProps) {
  const { state, setState, sessionScoped, campaignOptions, exportStatus, exportLoading, applyPreset, handleExport } = tg;

  return (
    <>
      <div className="page-header">
        <h1 className="page-header__title">{title}</h1>
        {freshness ? (
          <FreshnessBadge
            stale={freshness.stale}
            lagSeconds={freshness.lag_seconds ?? freshness.ch_lag_seconds}
          />
        ) : null}
        <p className="page-header__desc text-sm">{telegramPiiNotice()}</p>
      </div>

      <nav className="tab-bar" aria-label="Telegram reports">
        {TELEGRAM_REPORT_PAGES.map((page) => (
          <a
            key={page.path}
            href={telegramReportHref(page.path, state)}
            className={`tab-bar__item${pagePath === page.path ? ' tab-bar__item--active' : ''}`}
            aria-current={pagePath === page.path ? 'page' : undefined}
          >
            {page.label}
          </a>
        ))}
      </nav>

      <form
        className="filter-form settings-panel"
        onSubmit={(e) => {
          e.preventDefault();
          onSubmit();
        }}
      >
        {!sessionScoped ? (
          <FormField label="Customer ID" htmlFor="tg-report-customer">
            <input
              id="tg-report-customer"
              className="form-input"
              value={state.customerInput}
              placeholder="customer_id (UUID)"
              onChange={(e) => {
                setState((prev) => ({
                  ...prev,
                  customerInput: e.target.value,
                  campaignInput: '',
                }));
                void tg.refreshCampaignOptions();
              }}
            />
          </FormField>
        ) : null}

        <FormField label="Campaign" htmlFor="tg-report-campaign">
          {campaignOptions.length > 0 ? (
            <select
              id="tg-report-campaign"
              className="form-input"
              value={state.campaignInput}
              onChange={(e) => setState((prev) => ({ ...prev, campaignInput: e.target.value }))}
            >
              <option value="">All campaigns</option>
              {campaignOptions.map((c) => (
                <option key={c.id} value={c.id}>{c.name ?? c.id}</option>
              ))}
            </select>
          ) : (
            <input
              id="tg-report-campaign"
              className="form-input"
              value={state.campaignInput}
              placeholder="All campaigns (UUID)"
              onChange={(e) => setState((prev) => ({ ...prev, campaignInput: e.target.value }))}
            />
          )}
        </FormField>

        <div className="date-presets">
          <span className="date-presets__label">Range</span>
          {REPORT_DATE_PRESETS.map((preset) => (
            <button
              key={preset.id}
              type="button"
              className={`date-preset${state.activePreset === preset.id ? ' date-preset--active' : ''}`}
              onClick={() => {
                applyPreset(preset);
                onSubmit();
              }}
            >
              {preset.label}
            </button>
          ))}
        </div>

        <FormField label="From" htmlFor="tg-report-from">
          <input
            id="tg-report-from"
            className="form-input"
            value={state.from}
            onChange={(e) => setState((prev) => ({
              ...prev,
              from: e.target.value,
              activePreset: '',
            }))}
          />
        </FormField>
        <FormField label="To" htmlFor="tg-report-to">
          <input
            id="tg-report-to"
            className="form-input"
            value={state.to}
            onChange={(e) => setState((prev) => ({
              ...prev,
              to: e.target.value,
              activePreset: '',
            }))}
          />
        </FormField>

        <div className="toolbar-row">
          <Button
            label="Query"
            variant="primary"
            type="submit"
            loading={loading}
            disabled={loading}
          />
          {showExport ? (
            <Button
              label="Export CSV"
              variant="secondary"
              loading={exportLoading}
              disabled={loading || exportLoading}
              onClick={() => void handleExport()}
            />
          ) : null}
          {exportStatus ? (
            <span className="text-muted text-sm">{exportStatus}</span>
          ) : null}
        </div>
      </form>

      {loading ? <p className="loading-hint">Loading report…</p> : null}
      {!loading ? children : null}
    </>
  );
}

export function TelegramReportEmpty() {
  const copy = buyerEmptyCopy('reports_empty');
  return (
    <div className="empty-state section-block">
      <h2 className="empty-state__title">{copy.title}</h2>
      <p>{copy.description}</p>
      {copy.actionHref ? (
        <ButtonLink
          href={copy.actionHref}
          label={copy.actionLabel ?? 'Open'}
          variant="secondary"
        />
      ) : null}
    </div>
  );
}
