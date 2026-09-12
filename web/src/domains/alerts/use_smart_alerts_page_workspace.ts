import { useCallback, useMemo, useState } from 'react';
import { toast } from 'sonner';

import {
  ackSmartAlertEvent,
  createSmartAlertRule,
  deleteSmartAlertRule,
  listSmartAlertHistory,
  listSmartAlertRules,
  updateSmartAlertRule,
} from '@/api/smart_alerts_api';
import { useResource } from '@/api/use_resource';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import type { SmartAlertEvent, SmartAlertRule, SmartAlertRuleTemplate } from '@/api/types';
import { useSession } from '@/hooks/use_session';
import { exportHubErrorMessage } from '@/domains/exports/export_hub_errors';
import { confirmDestructiveAction } from '@/lib/mutation_audit';
import { sessionHasPermission } from '@/lib/session_permissions';
import {
  requireNonEmpty,
  toastValidationError,
  validationError,
  type AdminValidationError,
} from '@/lib/admin_validation_error';
import {
  SMART_ALERT_TEMPLATE_OPTIONS,
  buildUpsertSmartAlertRuleFromTemplate,
} from '@/domains/alerts/smart_alerts_templates';

const HISTORY_PAGE_SIZE = 25;

export type SmartAlertsDraft = {
  name: string;
  template: SmartAlertRuleTemplate | '';
  threshold: string;
  campaignId: string;
  webhookUrl: string;
  enabled: boolean;
};

const DEFAULT_DRAFT: SmartAlertsDraft = {
  name: '',
  template: 'budget_burn_pct',
  threshold: '80',
  campaignId: '',
  webhookUrl: '',
  enabled: true,
};

export function useSmartAlertsPageWorkspace() {
  const { session, user } = useSession();
  const [customerId, setCustomerId] = useState(session?.default_customer_id ?? '');
  const [draft, setDraft] = useState<SmartAlertsDraft>(DEFAULT_DRAFT);
  const [selectedRuleId, setSelectedRuleId] = useState<string | undefined>();
  const { refreshToken: rulesRefreshToken, bumpRefresh: bumpRulesRefresh } = useRefreshToken();
  const { refreshToken: historyRefreshToken, bumpRefresh: bumpHistoryRefresh } = useRefreshToken();
  const [historyPage, setHistoryPage] = useState(0);
  const [saving, setSaving] = useState(false);
  const [ackingEventId, setAckingEventId] = useState<string | undefined>();
  const [formValidationError, setFormValidationError] = useState<
    AdminValidationError | undefined
  >();

  const canManage = sessionHasPermission(user?.permissions, 'campaigns:write');
  const trimmedCustomerId = customerId.trim();

  const {
    data: rules,
    error: rulesError,
    fetching: rulesFetching,
  } = useResource(
    (signal) => {
      if (!trimmedCustomerId) {
        return Promise.resolve(undefined);
      }
      return listSmartAlertRules({ customer_id: trimmedCustomerId }, signal);
    },
    [trimmedCustomerId, rulesRefreshToken]
  );

  const {
    data: history,
    error: historyError,
    fetching: historyFetching,
  } = useResource(
    (signal) => {
      if (!trimmedCustomerId) {
        return Promise.resolve(undefined);
      }
      return listSmartAlertHistory(
        {
          customer_id: trimmedCustomerId,
          limit: HISTORY_PAGE_SIZE,
          offset: historyPage * HISTORY_PAGE_SIZE,
        },
        signal
      );
    },
    [trimmedCustomerId, historyPage, historyRefreshToken]
  );

  const selectedRule = useMemo(
    () => rules?.find((row) => row.id === selectedRuleId),
    [rules, selectedRuleId]
  );

  const refreshRules = useCoalescedBumpRefresh(bumpRulesRefresh, rulesFetching);
  const refreshHistory = useCoalescedBumpRefresh(bumpHistoryRefresh, historyFetching);

  const loadRuleIntoDraft = useCallback((rule: SmartAlertRule) => {
    setSelectedRuleId(rule.id);
    setDraft({
      name: rule.name ?? '',
      template: (rule.template as SmartAlertRuleTemplate | undefined) ?? '',
      threshold: rule.threshold != null ? String(rule.threshold) : '',
      campaignId: rule.campaign_id ?? '',
      webhookUrl: rule.webhook_url ?? '',
      enabled: Boolean(rule.enabled),
    });
    setFormValidationError(undefined);
  }, []);

  const resetDraft = useCallback(() => {
    setSelectedRuleId(undefined);
    setDraft(DEFAULT_DRAFT);
    setFormValidationError(undefined);
  }, []);

  const onSaveRule = useCallback(async () => {
    if (!canManage) {
      toast.error('campaigns:write permission required');
      return;
    }
    const customerIdCheck = requireNonEmpty(trimmedCustomerId, 'Customer ID', 'customer_id');
    if (!customerIdCheck.ok) {
      setFormValidationError(customerIdCheck.error);
      toastValidationError(customerIdCheck.error);
      return;
    }
    const templateCheck = requireNonEmpty(draft.template, 'Template', 'template');
    if (!templateCheck.ok) {
      setFormValidationError(templateCheck.error);
      toastValidationError(templateCheck.error);
      return;
    }
    const thresholdValue = Number.parseFloat(draft.threshold.trim());
    if (!Number.isFinite(thresholdValue)) {
      const err = validationError('Threshold must be a number.', { field: 'threshold' });
      setFormValidationError(err);
      toastValidationError(err);
      return;
    }
    const webhookCheck = requireNonEmpty(draft.webhookUrl, 'Webhook URL', 'webhook_url');
    if (!webhookCheck.ok) {
      setFormValidationError(webhookCheck.error);
      toastValidationError(webhookCheck.error);
      return;
    }
    const body = buildUpsertSmartAlertRuleFromTemplate(
      templateCheck.value as SmartAlertRuleTemplate,
      {
        customer_id: customerIdCheck.value,
        name: draft.name.trim() || undefined,
        threshold: thresholdValue,
        campaign_id: draft.campaignId.trim() || undefined,
        webhook_url: webhookCheck.value,
        enabled: draft.enabled,
      }
    );
    setSaving(true);
    try {
      if (selectedRuleId) {
        await updateSmartAlertRule(selectedRuleId, body);
        toast.success('Alert rule updated');
      } else {
        const created = await createSmartAlertRule(body);
        if (created.id) {
          setSelectedRuleId(created.id);
        }
        toast.success('Alert rule created');
      }
      refreshRules();
      setFormValidationError(undefined);
    } catch (err: unknown) {
      toast.error(exportHubErrorMessage(err));
    } finally {
      setSaving(false);
    }
  }, [canManage, draft, refreshRules, selectedRuleId, trimmedCustomerId]);

  const onDeleteRule = useCallback(async () => {
    if (!canManage || !selectedRuleId) {
      return;
    }
    const rule = rules?.find((row) => row.id === selectedRuleId);
    if (!rule) {
      return;
    }
    if (!confirmDestructiveAction(`Delete alert rule "${rule.name}"?`)) {
      return;
    }
    setSaving(true);
    try {
      await deleteSmartAlertRule(selectedRuleId);
      resetDraft();
      refreshRules();
      toast.success('Alert rule deleted');
    } catch (err: unknown) {
      toast.error(exportHubErrorMessage(err));
    } finally {
      setSaving(false);
    }
  }, [canManage, refreshRules, resetDraft, rules, selectedRuleId]);

  const onToggleRule = useCallback(
    async (rule: SmartAlertRule) => {
      if (!canManage || !rule.id) {
        return;
      }
      setSaving(true);
      try {
        await updateSmartAlertRule(rule.id, {
          customer_id: rule.customer_id ?? trimmedCustomerId,
          name: rule.name ?? '',
          template: rule.template,
          metric: rule.metric,
          operator: rule.operator,
          threshold: rule.threshold ?? 0,
          window_minutes: rule.window_minutes,
          campaign_id: rule.campaign_id,
          webhook_url: rule.webhook_url ?? '',
          enabled: !rule.enabled,
        });
        refreshRules();
        toast.success(rule.enabled ? 'Alert rule disabled' : 'Alert rule enabled');
      } catch (err: unknown) {
        toast.error(exportHubErrorMessage(err));
      } finally {
        setSaving(false);
      }
    },
    [canManage, refreshRules, trimmedCustomerId]
  );

  const onAckEvent = useCallback(
    async (eventId: string) => {
      if (!canManage) {
        toast.error('campaigns:write permission required');
        return;
      }
      setAckingEventId(eventId);
      try {
        await ackSmartAlertEvent(eventId);
        refreshHistory();
        toast.success('Alert acknowledged');
      } catch (err: unknown) {
        toast.error(exportHubErrorMessage(err));
      } finally {
        setAckingEventId(undefined);
      }
    },
    [canManage, refreshHistory]
  );

  return {
    customerId,
    setCustomerId,
    draft,
    setDraft,
    selectedRuleId,
    setSelectedRuleId,
    rules: rules ?? [],
    rulesHasSnapshot: rules != null,
    rulesError,
    rulesFetching,
    history: history ?? [],
    historyHasSnapshot: history != null,
    historyError,
    historyFetching,
    historyPage,
    setHistoryPage,
    historyPageSize: HISTORY_PAGE_SIZE,
    canManage,
    saving,
    ackingEventId,
    formValidationError,
    templateOptions: SMART_ALERT_TEMPLATE_OPTIONS,
    selectedRule,
    resetDraft,
    loadRuleIntoDraft,
    onSaveRule,
    onDeleteRule,
    onToggleRule,
    onAckEvent,
  };
}
