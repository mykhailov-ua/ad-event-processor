import { useCallback, useEffect, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { toast } from 'sonner';

import {
  getCampaignWizardSession,
  listCampaignOnboardingTemplates,
  postCampaignWizardSession,
} from '@/api/campaigns_api';
import { ApiError } from '@/api/client';
import type { CampaignOnboardingTemplate, CampaignWizardSessionRequest } from '@/api/types';
import { PrimaryActionButton, SecondaryActionButton } from '@/components/system/action_buttons';
import { ErrorBlock } from '@/components/system/error_block';
import { StubBanner } from '@/components/system/stub_banner';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { JsonDashboardView } from '@/domains/dashboards/json_dashboard_view';
import { useResource } from '@/hooks/use_resource';
import { useSession } from '@/hooks/use_session';

type TemplateKey = NonNullable<CampaignWizardSessionRequest['template_key']>;

function panelError(error: Error, title: string) {
  if (error instanceof ApiError && error.status === 501) {
    return <StubBanner title={`${title} unavailable`} message={error.message} />;
  }
  return <ErrorBlock title={title} message={error.message} />;
}

export function CampaignWizardPanel() {
  const [searchParams] = useSearchParams();
  const { session } = useSession();
  const defaultCustomerId =
    searchParams.get('customer_id') ?? session?.default_customer_id ?? '';

  const templatesResource = useResource(listCampaignOnboardingTemplates, []);

  const [draftCustomerId, setDraftCustomerId] = useState(defaultCustomerId);
  const [draftTemplateKey, setDraftTemplateKey] = useState<TemplateKey | ''>('');
  const [draftSessionId, setDraftSessionId] = useState('');
  const [sessionPayload, setSessionPayload] = useState<Record<string, unknown> | undefined>();
  const [creating, setCreating] = useState(false);
  const [committing, setCommitting] = useState(false);
  const [polling, setPolling] = useState(false);
  const [actionError, setActionError] = useState<Error | undefined>();
  const [pollToken, setPollToken] = useState(0);
  const [createOpen, setCreateOpen] = useState(false);
  const [sessionOpen, setSessionOpen] = useState(false);

  const templates = useMemo(
    () => templatesResource.data ?? [],
    [templatesResource.data],
  );

  useEffect(() => {
    if (templates.length === 0 || draftTemplateKey) {
      return;
    }
    setDraftTemplateKey(templates[0]?.key as TemplateKey);
  }, [draftTemplateKey, templates]);

  const selectedTemplate = useMemo(
    () => templates.find((item) => item.key === draftTemplateKey),
    [draftTemplateKey, templates],
  );

  const sessionResource = useResource(
    (signal) => {
      if (!draftSessionId.trim()) {
        return Promise.resolve(undefined);
      }
      return getCampaignWizardSession(draftSessionId.trim(), signal);
    },
    [draftSessionId, pollToken],
  );

  const onCreateSession = useCallback(async () => {
    const customerId = draftCustomerId.trim();
    if (!customerId) {
      setActionError(new Error('Customer ID is required.'));
      return;
    }
    if (!draftTemplateKey) {
      setActionError(new Error('Template is required.'));
      return;
    }
    setCreating(true);
    setActionError(undefined);
    try {
      const result = await postCampaignWizardSession({
        action: 'create',
        customer_id: customerId,
        template_key: draftTemplateKey,
      });
      if ('session_id' in result && result.session_id) {
        setDraftSessionId(result.session_id);
        setSessionPayload(result as unknown as Record<string, unknown>);
        setPollToken((value) => value + 1);
        toast.success('Wizard session created');
        setCreateOpen(false);
      } else {
        setSessionPayload(result as unknown as Record<string, unknown>);
      }
    } catch (err) {
      setActionError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setCreating(false);
    }
  }, [draftCustomerId, draftTemplateKey]);

  const onPollSession = useCallback(() => {
    if (!draftSessionId.trim()) {
      return;
    }
    setPolling(true);
    setPollToken((value) => value + 1);
    setPolling(false);
    toast.success('Wizard session polled');
  }, [draftSessionId]);

  const onCommitSession = useCallback(async () => {
    const sessionId = draftSessionId.trim();
    if (!sessionId) {
      setActionError(new Error('Session ID is required to commit.'));
      return;
    }
    setCommitting(true);
    setActionError(undefined);
    try {
      const result = await postCampaignWizardSession({
        action: 'commit',
        session_id: sessionId,
        idempotency_key: crypto.randomUUID(),
      });
      setSessionPayload(result as unknown as Record<string, unknown>);
      toast.success('Wizard session committed');
      setSessionOpen(false);
    } catch (err) {
      setActionError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setCommitting(false);
    }
  }, [draftSessionId]);

  const displayed = sessionResource.data ?? sessionPayload;

  return (
    <section className="ui-filter-panel">
      <h2 className="text-base font-semibold">Onboarding wizard</h2>

      {templatesResource.error
        ? panelError(templatesResource.error, 'Could not load onboarding templates')
        : null}

      <PrimaryActionButton
        disabled={!draftTemplateKey}
        loading={creating}
        onClick={() => setCreateOpen(true)}
        type="button"
      >
        Create session
      </PrimaryActionButton>
      <SecondaryActionButton onClick={() => setSessionOpen(true)} type="button">
        Manage session
      </SecondaryActionButton>
      {draftSessionId.trim() ? (
        <p className="text-sm text-muted-foreground font-mono">Session: {draftSessionId}</p>
      ) : null}
      <Dialog onOpenChange={setCreateOpen} open={createOpen}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Create wizard session</DialogTitle>
          </DialogHeader>
          <div className="grid gap-4">
            <div className="grid gap-2">
              <Label htmlFor="wizard-customer-id">Customer ID</Label>
              <Input
                id="wizard-customer-id"
                value={draftCustomerId}
                onChange={(event) => setDraftCustomerId(event.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="wizard-template-key">Template</Label>
              <Select
                value={draftTemplateKey}
                onValueChange={(value) => setDraftTemplateKey(value as TemplateKey)}
              >
                <SelectTrigger id="wizard-template-key" className="h-9 w-full text-sm">
                  <SelectValue placeholder="Select template" />
                </SelectTrigger>
                <SelectContent>
                  {templates.map((template: CampaignOnboardingTemplate) => (
                    <SelectItem key={template.key} value={template.key ?? ''}>
                      {template.title ?? template.key}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
          <DialogFooter>
            <PrimaryActionButton
              disabled={!draftTemplateKey}
              loading={creating}
              onClick={onCreateSession}
              type="button"
            >
              Create
            </PrimaryActionButton>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog onOpenChange={setSessionOpen} open={sessionOpen}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Manage wizard session</DialogTitle>
          </DialogHeader>
          <div className="grid gap-4">
            <div className="grid gap-2">
              <Label htmlFor="wizard-session-id">Session ID</Label>
              <Input
                id="wizard-session-id"
                value={draftSessionId}
                onChange={(event) => setDraftSessionId(event.target.value)}
              />
            </div>
          </div>
          <DialogFooter className="gap-2 sm:justify-end">
            <SecondaryActionButton
              disabled={!draftSessionId.trim()}
              loading={polling}
              onClick={onPollSession}
              type="button"
            >
              Poll
            </SecondaryActionButton>
            <PrimaryActionButton
              disabled={!draftSessionId.trim()}
              loading={committing}
              onClick={onCommitSession}
              type="button"
            >
              Commit
            </PrimaryActionButton>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {selectedTemplate ? (
        <div className="ui-surface rounded-xl p-3 text-sm">
          <p className="font-medium">{selectedTemplate.title}</p>
          <p className="text-muted-foreground">{selectedTemplate.description}</p>
          <p className="text-muted-foreground">Traffic family: {selectedTemplate.traffic_family}</p>
        </div>
      ) : null}

      {actionError ? panelError(actionError, 'Wizard action failed') : null}
      {sessionResource.error ? panelError(sessionResource.error, 'Could not load wizard session') : null}

      {displayed ? (
        <JsonDashboardView payload={displayed as Record<string, unknown>} />
      ) : null}
    </section>
  );
}
