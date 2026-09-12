// campaign integration tab: template apply, dry-run preview, copy URLs, on-demand health.
import { useCallback, useEffect, useState } from 'react';
import { toast } from 'sonner';

import {
  applyCampaignTemplates,
  dryRunCampaignTemplates,
  getCampaignIntegrationHealth,
  listCampaignOutboundPostbacks,
  listCampaignStatusSchemes,
  replaceCampaignOutboundPostbacks,
  replaceCampaignStatusSchemes,
  testCampaignOutboundPostback,
} from '@/api/campaigns_api';
import type {
  OutboundPostback,
  OutboundPostbackWrite,
  StatusSchemeRule,
} from '@/api/campaigns_types';
import type {
  ApplyCampaignTemplatesResult,
  Campaign,
  CampaignIntegrationHealth,
  DryRunCampaignTemplatesResult,
} from '@/api/types';
type StatusSchemeDraft = {
  when_status: string;
  when_goal: string;
  set_internal_status: string;
  set_goal_name: string;
  payout_mode: string;
  payout_micro: string;
  fire_outbound: boolean;
  enabled: boolean;
};

function ruleToDraft(rule: StatusSchemeRule): StatusSchemeDraft {
  return {
    when_status: rule.when_status ?? '',
    when_goal: rule.when_goal ?? '',
    set_internal_status: rule.set_internal_status ?? '',
    set_goal_name: rule.set_goal_name ?? '',
    payout_mode: rule.payout_mode ?? 'inherit',
    payout_micro: rule.payout_micro != null ? String(rule.payout_micro) : '',
    fire_outbound: rule.fire_outbound ?? true,
    enabled: rule.enabled ?? true,
  };
}

function draftsToRules(drafts: StatusSchemeDraft[]): StatusSchemeRule[] {
  const out: StatusSchemeRule[] = [];
  drafts.forEach((draft, index) => {
    const whenStatus = draft.when_status.trim();
    const whenGoal = draft.when_goal.trim();
    if (!whenStatus && !whenGoal) {
      return;
    }
    const rule: StatusSchemeRule = {
      sort_order: index + 1,
      when_status: whenStatus || undefined,
      when_goal: whenGoal || undefined,
      set_internal_status: draft.set_internal_status.trim() || undefined,
      set_goal_name: draft.set_goal_name.trim() || undefined,
      payout_mode: (draft.payout_mode.trim() || 'inherit') as StatusSchemeRule['payout_mode'],
      fire_outbound: draft.fire_outbound,
      enabled: draft.enabled,
    };
    const payout = draft.payout_micro.trim();
    if (payout) {
      const parsed = Number(payout);
      if (!Number.isFinite(parsed) || parsed < 0) {
        throw new Error('Payout micro must be a non-negative number');
      }
      rule.payout_micro = Math.trunc(parsed);
    }
    out.push(rule);
  });
  return out;
}

export function useCampaignIntegrationPanelWorkspace(campaignId: string, campaign?: Campaign) {
  const [draftTrafficSource, setDraftTrafficSource] = useState('');
  const [draftAffiliateNetwork, setDraftAffiliateNetwork] = useState('');
  const [draftTrackingDomain, setDraftTrackingDomain] = useState('');
  const [applying, setApplying] = useState(false);
  const [dryRunning, setDryRunning] = useState(false);
  const [applyResult, setApplyResult] = useState<ApplyCampaignTemplatesResult | undefined>();
  const [dryRunResult, setDryRunResult] = useState<DryRunCampaignTemplatesResult | undefined>();
  const [applyError, setApplyError] = useState<Error | undefined>();
  const [dryRunError, setDryRunError] = useState<Error | undefined>();
  const [health, setHealth] = useState<CampaignIntegrationHealth | undefined>();
  const [healthError, setHealthError] = useState<Error | undefined>();
  const [healthLoading, setHealthLoading] = useState(false);
  const [schemeDrafts, setSchemeDrafts] = useState<StatusSchemeDraft[]>([]);
  const [schemeLoading, setSchemeLoading] = useState(false);
  const [schemeSaving, setSchemeSaving] = useState(false);
  const [schemeLoaded, setSchemeLoaded] = useState(false);
  const [schemeError, setSchemeError] = useState<Error | undefined>();
  const [schemeSaveSuccess, setSchemeSaveSuccess] = useState(false);
  const [outboundDrafts, setOutboundDrafts] = useState<OutboundPostbackDraft[]>([]);
  const [outboundLoading, setOutboundLoading] = useState(false);
  const [outboundSaving, setOutboundSaving] = useState(false);
  const [outboundLoaded, setOutboundLoaded] = useState(false);
  const [outboundError, setOutboundError] = useState<Error | undefined>();
  const [outboundSaveSuccess, setOutboundSaveSuccess] = useState(false);
  const [outboundTestMessage, setOutboundTestMessage] = useState<string | undefined>();

  useEffect(() => {
    setApplyResult(undefined);
    setDryRunResult(undefined);
    setApplyError(undefined);
    setDryRunError(undefined);
    setHealth(undefined);
    setHealthError(undefined);
    setSchemeDrafts([]);
    setSchemeLoaded(false);
    setSchemeError(undefined);
    setSchemeSaveSuccess(false);
    setOutboundDrafts([]);
    setOutboundLoaded(false);
    setOutboundError(undefined);
    setOutboundSaveSuccess(false);
    setOutboundTestMessage(undefined);
  }, [campaignId]);

  const buildApplyBody = useCallback(() => {
    const body: Parameters<typeof applyCampaignTemplates>[1] = {};
    if (draftTrafficSource.trim()) {
      body.traffic_source = draftTrafficSource.trim();
    }
    if (draftAffiliateNetwork.trim()) {
      body.affiliate_network = draftAffiliateNetwork.trim();
    }
    if (draftTrackingDomain.trim()) {
      body.tracking_domain = draftTrackingDomain.trim();
    }
    return body;
  }, [draftAffiliateNetwork, draftTrackingDomain, draftTrafficSource]);

  const onLoadHealth = useCallback(async () => {
    setHealthLoading(true);
    setHealthError(undefined);
    try {
      const result = await getCampaignIntegrationHealth(campaignId);
      setHealth(result);
    } catch (err: unknown) {
      setHealthError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setHealthLoading(false);
    }
  }, [campaignId]);

  const onDryRunTemplates = useCallback(async () => {
    setDryRunning(true);
    setDryRunError(undefined);
    setDryRunResult(undefined);
    try {
      const result = await dryRunCampaignTemplates(campaignId, buildApplyBody());
      setDryRunResult(result);
    } catch (err: unknown) {
      setDryRunError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setDryRunning(false);
    }
  }, [buildApplyBody, campaignId]);

  const onLoadOutboundPostbacks = useCallback(async () => {
    setOutboundLoading(true);
    setOutboundError(undefined);
    setOutboundSaveSuccess(false);
    setOutboundTestMessage(undefined);
    try {
      const result = await listCampaignOutboundPostbacks(campaignId);
      const rows = result.postbacks ?? [];
      setOutboundDrafts(rows.length > 0 ? rows.map(outboundToDraft) : [emptyOutboundDraft()]);
      setOutboundLoaded(true);
    } catch (err: unknown) {
      setOutboundError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setOutboundLoading(false);
    }
  }, [campaignId]);

  const onSaveOutboundPostbacks = useCallback(async () => {
    setOutboundSaving(true);
    setOutboundError(undefined);
    setOutboundSaveSuccess(false);
    try {
      const postbacks = draftsToOutboundPostbacks(outboundDrafts);
      await replaceCampaignOutboundPostbacks(campaignId, { postbacks });
      setOutboundSaveSuccess(true);
      toast.success('Outbound postbacks saved');
      await onLoadOutboundPostbacks();
    } catch (err: unknown) {
      setOutboundError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setOutboundSaving(false);
    }
  }, [campaignId, onLoadOutboundPostbacks, outboundDrafts]);

  const onTestOutboundPostback = useCallback(
    async (postbackId: string) => {
      setOutboundError(undefined);
      setOutboundTestMessage(undefined);
      try {
        const result = await testCampaignOutboundPostback(campaignId, postbackId);
        if (result.ok) {
          setOutboundTestMessage(
            result.rendered_url ? `Dry-run OK: ${result.rendered_url}` : 'Dry-run OK'
          );
        } else {
          setOutboundTestMessage(
            result.error ? `Dry-run failed: ${result.error}` : 'Dry-run failed'
          );
        }
      } catch (err: unknown) {
        setOutboundError(err instanceof Error ? err : new Error(String(err)));
      }
    },
    [campaignId]
  );

  const onLoadStatusSchemes = useCallback(async () => {
    setSchemeLoading(true);
    setSchemeError(undefined);
    setSchemeSaveSuccess(false);
    try {
      const result = await listCampaignStatusSchemes(campaignId);
      const rules = result.rules ?? [];
      setSchemeDrafts(rules.length > 0 ? rules.map(ruleToDraft) : [emptySchemeDraft()]);
      setSchemeLoaded(true);
    } catch (err: unknown) {
      setSchemeError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setSchemeLoading(false);
    }
  }, [campaignId]);

  const onSaveStatusSchemes = useCallback(async () => {
    setSchemeSaving(true);
    setSchemeError(undefined);
    setSchemeSaveSuccess(false);
    try {
      const rules = draftsToRules(schemeDrafts);
      await replaceCampaignStatusSchemes(campaignId, { rules });
      setSchemeSaveSuccess(true);
      toast.success('Status scheme saved');
      await onLoadStatusSchemes();
    } catch (err: unknown) {
      setSchemeError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setSchemeSaving(false);
    }
  }, [campaignId, onLoadStatusSchemes, schemeDrafts]);

  const onApplyTemplates = useCallback(async () => {
    setApplying(true);
    setApplyError(undefined);
    setApplyResult(undefined);
    try {
      const result = await applyCampaignTemplates(campaignId, buildApplyBody());
      setApplyResult(result);
      toast.success('Integration templates applied');
    } catch (err: unknown) {
      setApplyError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setApplying(false);
    }
  }, [buildApplyBody, campaignId]);

  const clickCopyURL =
    applyResult?.traffic_source?.target_url ??
    dryRunResult?.target_url ??
    campaign?.target_url ??
    '';
  const postbackCopyURL =
    applyResult?.affiliate_postback?.panel_postback_url ??
    dryRunResult?.panel_postback_url ??
    applyResult?.affiliate_postback?.url_template ??
    dryRunResult?.postback_url_template ??
    '';

  return {
    draftTrafficSource,
    setDraftTrafficSource,
    draftAffiliateNetwork,
    setDraftAffiliateNetwork,
    draftTrackingDomain,
    setDraftTrackingDomain,
    applying,
    dryRunning,
    applyResult,
    dryRunResult,
    applyError,
    dryRunError,
    health,
    healthError,
    healthLoading,
    clickCopyURL,
    postbackCopyURL,
    statusIntegrationSchemaName: campaign?.status_integration_schema_name,
    onLoadHealth,
    onApplyTemplates,
    onDryRunTemplates,
    schemeDrafts,
    setSchemeDrafts,
    schemeLoading,
    schemeSaving,
    schemeLoaded,
    schemeError,
    schemeSaveSuccess,
    onLoadStatusSchemes,
    onSaveStatusSchemes,
    outboundDrafts,
    setOutboundDrafts,
    outboundLoading,
    outboundSaving,
    outboundLoaded,
    outboundError,
    outboundSaveSuccess,
    outboundTestMessage,
    onLoadOutboundPostbacks,
    onSaveOutboundPostbacks,
    onTestOutboundPostback,
  };
}

type OutboundPostbackDraft = {
  id?: string;
  name: string;
  priority: string;
  enabled: boolean;
  provider: string;
  url_template: string;
  api_token: string;
  target_event: string;
  trigger_kind: string;
  trigger_value: string;
  test_event_code: string;
  sample_percent: string;
  delay_seconds: string;
};

function outboundToDraft(row: OutboundPostback): OutboundPostbackDraft {
  return {
    id: row.id,
    name: row.name ?? '',
    priority: row.priority != null ? String(row.priority) : '0',
    enabled: row.enabled ?? true,
    provider: row.provider ?? 'webhook',
    url_template: row.url_template ?? '',
    api_token: '',
    target_event: row.target_event ?? 'conversion',
    trigger_kind: row.trigger_kind ?? 'conversion',
    trigger_value: row.trigger_value ?? '',
    test_event_code: row.test_event_code ?? '',
    sample_percent: row.sample_percent != null ? String(row.sample_percent) : '100',
    delay_seconds: row.delay_seconds != null ? String(row.delay_seconds) : '0',
  };
}

function draftsToOutboundPostbacks(drafts: OutboundPostbackDraft[]): OutboundPostbackWrite[] {
  const out: OutboundPostbackWrite[] = [];
  drafts.forEach((draft, index) => {
    const url = draft.url_template.trim();
    if (!url) {
      return;
    }
    const row: OutboundPostbackWrite = {
      name: draft.name.trim(),
      priority: Number(draft.priority.trim() || String(index)),
      enabled: draft.enabled,
      provider: draft.provider.trim() || 'webhook',
      url_template: url,
      target_event: draft.target_event.trim() || 'conversion',
      trigger_kind: (draft.trigger_kind.trim() ||
        'conversion') as OutboundPostbackWrite['trigger_kind'],
      trigger_value: draft.trigger_value.trim(),
      test_event_code: draft.test_event_code.trim(),
      sample_percent: Number(draft.sample_percent.trim() || '100'),
      delay_seconds: Number(draft.delay_seconds.trim() || '0'),
    };
    if (draft.api_token.trim()) {
      row.api_token = draft.api_token.trim();
    }
    out.push(row);
  });
  return out;
}

function emptyOutboundDraft(): OutboundPostbackDraft {
  return {
    name: '',
    priority: '0',
    enabled: true,
    provider: 'webhook',
    url_template: '',
    api_token: '',
    target_event: 'conversion',
    trigger_kind: 'conversion',
    trigger_value: '',
    test_event_code: '',
    sample_percent: '100',
    delay_seconds: '0',
  };
}

function emptySchemeDraft(): StatusSchemeDraft {
  return {
    when_status: '',
    when_goal: '',
    set_internal_status: '',
    set_goal_name: '',
    payout_mode: 'inherit',
    payout_micro: '',
    fire_outbound: true,
    enabled: true,
  };
}

export type CampaignIntegrationPanelWorkspace = ReturnType<
  typeof useCampaignIntegrationPanelWorkspace
>;
