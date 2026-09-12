import { useCallback, useEffect, useState } from 'react';
import { toast } from 'sonner';

import { isAbortError } from '@/api/client';
import { createReportAutomationRule } from '@/api/reports_api';
import type { SourceQualityRow } from '@/api/types';
import {
  buildSourceQualityFilterSnapshot,
  defaultReportRuleActionForRow,
  defaultReportRuleName,
  resolveReportRuleCampaignId,
  type ReportRuleAction,
  type ReportRuleFilterContext,
} from '@/lib/report_rule_snapshot';
import { toError } from '@/lib/admin_error';
import {
  actionGuardError,
  requireNonEmpty,
  toastValidationError,
} from '@/lib/admin_validation_error';

export type UseReportRuleCreateDialogWorkspaceArgs = {
  open: boolean;
  reportKey: string;
  row: SourceQualityRow | undefined;
  filterContext: ReportRuleFilterContext;
  onCreated?: () => void;
};

export function useReportRuleCreateDialogWorkspace({
  open,
  reportKey,
  row,
  filterContext,
  onCreated,
}: UseReportRuleCreateDialogWorkspaceArgs) {
  const [action, setAction] = useState<ReportRuleAction>('blacklist_placement');
  const [name, setName] = useState('');
  const [threshold, setThreshold] = useState('');
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState<Error | undefined>();

  useEffect(() => {
    if (!open || !row) {
      return;
    }
    const defaultAction = defaultReportRuleActionForRow(row);
    setAction(defaultAction);
    setName(defaultReportRuleName(reportKey, row, defaultAction));
    setThreshold('');
    setCreateError(undefined);
  }, [open, reportKey, row]);

  const campaignId = row ? resolveReportRuleCampaignId(row, filterContext) : undefined;
  const canSubmit = Boolean(filterContext.customerId.trim() && campaignId && row && !creating);

  const onCreate = useCallback(async () => {
    if (!row || creating) {
      return;
    }
    const customerCheck = requireNonEmpty(filterContext.customerId, 'Customer', 'customer_id');
    if (!customerCheck.ok) {
      toastValidationError(customerCheck.error);
      return;
    }
    if (!campaignId) {
      const err = actionGuardError('Campaign ID is required on the row or in report filters.');
      toastValidationError(err);
      return;
    }
    setCreating(true);
    setCreateError(undefined);
    try {
      const snapshot = buildSourceQualityFilterSnapshot(row, filterContext);
      const parsedThreshold = threshold.trim() ? Number(threshold) : undefined;
      await createReportAutomationRule({
        customer_id: filterContext.customerId.trim(),
        campaign_id: campaignId,
        report_key: reportKey,
        name: name.trim() || defaultReportRuleName(reportKey, row, action),
        action,
        threshold: parsedThreshold,
        filter_snapshot: snapshot,
      });
      toast.success('Automation rule created');
      onCreated?.();
    } catch (err: unknown) {
      if (isAbortError(err)) {
        return;
      }
      setCreateError(toError(err));
    } finally {
      setCreating(false);
    }
  }, [action, campaignId, creating, filterContext, name, onCreated, reportKey, row, threshold]);

  return {
    action,
    onActionChange: setAction,
    name,
    onNameChange: setName,
    threshold,
    onThresholdChange: setThreshold,
    creating,
    createError,
    canSubmit,
    campaignId,
    onCreate,
  };
}
