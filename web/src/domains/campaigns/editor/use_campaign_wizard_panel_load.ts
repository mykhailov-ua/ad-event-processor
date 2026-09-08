// wizard panel load: templates when enabled; session GET by sessionId with pollToken; localSession seeds UI before first poll 2xx.
import { useCallback, useState } from 'react';

import { getCampaignWizardSession, listCampaignOnboardingTemplates } from '@/api/campaigns_api';
import type { CampaignOnboardingTemplate, CampaignWizardSession } from '@/api/types';
import { useResource } from '@/api/use_resource';

export function useCampaignWizardPanelLoad(enabled: boolean) {
  const [sessionId, setSessionId] = useState('');
  const [localSession, setLocalSession] = useState<CampaignWizardSession | undefined>();
  const [pollToken, setPollToken] = useState(0);

  const templatesResource = useResource(
    (signal) => {
      if (!enabled) {
        return Promise.resolve(undefined);
      }
      return listCampaignOnboardingTemplates(signal);
    },
    [enabled]
  );

  const sessionResource = useResource(
    (signal) => {
      if (!enabled || !sessionId.trim()) {
        return Promise.resolve(undefined);
      }
      return getCampaignWizardSession(sessionId.trim(), signal);
    },
    [enabled, pollToken, sessionId]
  );

  const bumpSessionPoll = useCallback(() => {
    setPollToken((value) => value + 1);
  }, []);

  const onSessionCreated = useCallback((id: string, session: CampaignWizardSession) => {
    setSessionId(id);
    setLocalSession(session);
    setPollToken((value) => value + 1);
  }, []);

  const onSessionUpdated = useCallback((session: CampaignWizardSession) => {
    setLocalSession(session);
    setPollToken((value) => value + 1);
  }, []);

  const onSessionCommitted = useCallback(() => {
    setSessionId('');
    setLocalSession(undefined);
  }, []);

  const onStartAnother = useCallback(() => {
    setSessionId('');
    setLocalSession(undefined);
  }, []);

  const resetSession = useCallback(() => {
    setSessionId('');
    setLocalSession(undefined);
    setPollToken(0);
  }, []);

  return {
    templates: (templatesResource.data ?? []) as CampaignOnboardingTemplate[],
    templatesError: templatesResource.error,
    templatesFetching: templatesResource.fetching,
    session: sessionResource.data ?? localSession,
    sessionError: sessionResource.error,
    sessionFetching: sessionResource.fetching,
    sessionId,
    onSessionCreated,
    onSessionUpdated,
    onSessionCommitted,
    onStartAnother,
    resetSession,
    bumpSessionPoll,
  };
}

export type CampaignWizardPanelLoad = ReturnType<typeof useCampaignWizardPanelLoad>;
