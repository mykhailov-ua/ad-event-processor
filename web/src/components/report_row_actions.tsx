import { useState } from 'react';
import { Link } from 'react-router-dom';
import { to } from '../lib/to.js';
import { can } from '../helpers/permissions.js';
import * as auth from '../helpers/auth.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import type { ReportRow } from '../types/api/report.js';
import {
  blockReportSource,
  costSyncDrillHref,
  pauseReportCampaign,
  reportRowCampaignId,
  reportRowPlacementId,
  smartAlertPrefillHref,
  type ReportRowActionContext,
} from '../helpers/report_row_actions.js';
import { Button, ButtonLink } from './button.js';

export type ReportRowActionsProps = {
  row: ReportRow;
  customerId?: string;
  reportEndpoint?: string;
};

function actionContext(row: ReportRow, customerId?: string): ReportRowActionContext {
  const record = row as Record<string, unknown>;
  const ivt = record.ivt_rate;
  return {
    customerId,
    campaignId: reportRowCampaignId(record),
    placementId: reportRowPlacementId(record),
    sub1: typeof record.sub1 === 'string' ? record.sub1 : undefined,
    sub2: typeof record.sub2 === 'string' ? record.sub2 : undefined,
    ivtRate: typeof ivt === 'number' ? ivt : undefined,
    spendMicro: typeof record.spend_micro === 'number' ? record.spend_micro : undefined,
  };
}

/**
 * Row-level buyer actions for live reports (pause, alert, block source).
 */
export function ReportRowActions({ row, customerId, reportEndpoint }: ReportRowActionsProps) {
  const user = auth.getUser();
  const perms = user?.permissions ?? [];
  const canWrite = can(perms, 'campaigns:write') || can(perms, 'campaigns:pause');
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const ctx = actionContext(row, customerId);
  const hasCampaign = Boolean(ctx.campaignId);
  const blockTarget = ctx.placementId || ctx.sub1;

  const run = async (fn: () => Promise<void>, okTitle: string) => {
    setBusy(true);
    const [, err] = await to(fn());
    setBusy(false);
    setOpen(false);
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      const view = mapServiceError(err);
      pushToastMessage({ title: view.title, message: view.message, code: view.code });
      return;
    }
    pushToastMessage({ title: okTitle, message: '' });
  };

  if (!hasCampaign && reportEndpoint !== 'discrepancy-buy-sell') {
    return <span className="text-muted text-xs">—</span>;
  }

  return (
    <div className="reports-hub__actions" data-testid="report-row-actions">
      <Button
        label="Actions"
        variant="ghost"
        size="sm"
        action="menu"
        disabled={busy}
        data-testid="report-row-actions-toggle"
        onClick={() => setOpen((v) => !v)}
      />
      {open ? (
        <div className="stack stack--sm mt-1" role="menu">
          {hasCampaign && canWrite ? (
            <Button
              label="Pause campaign"
              variant="secondary"
              size="sm"
              action="pause"
              disabled={busy}
              data-testid="report-action-pause"
              onClick={() => void run(() => pauseReportCampaign(ctx.campaignId!), 'Campaign paused')}
            />
          ) : null}
          {customerId ? (
            <ButtonLink
              href={smartAlertPrefillHref(ctx)}
              label="Create alert…"
              variant="secondary"
              size="sm"
              data-testid="report-action-alert"
            />
          ) : null}
          {hasCampaign && blockTarget && canWrite ? (
            <Button
              label={ctx.placementId ? 'Block placement' : 'Block sub'}
              variant="secondary"
              size="sm"
              action="block"
              disabled={busy}
              data-testid="report-action-block"
              onClick={() => void run(
                () => blockReportSource(ctx),
                ctx.placementId ? 'Placement block queued' : 'Source block queued',
              )}
            />
          ) : null}
          {reportEndpoint === 'discrepancy-buy-sell' && hasCampaign ? (
            <Link
              className="btn btn--secondary btn--sm"
              data-testid="report-action-cost-sync"
              to={costSyncDrillHref(ctx.campaignId!, customerId)}
            >
              Cost sync history
            </Link>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}
