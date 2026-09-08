// import panel load: sources when enabled; validate job polled via pollToken bump (draftJobId from enqueue or manual input).
import { useCallback, useState } from 'react';

import { getCampaignImportValidateJob, listMigrationSources } from '@/api/campaigns_api';
import { useResource } from '@/api/use_resource';

export function useCampaignImportPanelLoad(enabled: boolean) {
  const [draftJobId, setDraftJobId] = useState('');
  const [pollToken, setPollToken] = useState(0);

  const sourcesResource = useResource(
    (signal) => {
      if (!enabled) {
        return Promise.resolve(undefined);
      }
      return listMigrationSources(signal);
    },
    [enabled]
  );

  const jobResource = useResource(
    (signal) => {
      if (!enabled || !draftJobId.trim()) {
        return Promise.resolve(undefined);
      }
      return getCampaignImportValidateJob(draftJobId.trim(), signal);
    },
    [draftJobId, enabled, pollToken]
  );

  const pollJob = useCallback(() => {
    if (!draftJobId.trim()) {
      return;
    }
    setPollToken((value) => value + 1);
  }, [draftJobId]);

  const onJobEnqueued = useCallback((jobId: string) => {
    setDraftJobId(jobId);
    setPollToken((value) => value + 1);
  }, []);

  const resetJobLane = useCallback(() => {
    setDraftJobId('');
    setPollToken(0);
  }, []);

  return {
    sources: sourcesResource.data,
    sourcesError: sourcesResource.error,
    sourcesFetching: sourcesResource.fetching,
    job: jobResource.data,
    jobError: jobResource.error,
    jobFetching: jobResource.fetching,
    draftJobId,
    setDraftJobId,
    pollJob,
    onJobEnqueued,
    resetJobLane,
  };
}

export type CampaignImportPanelLoad = ReturnType<typeof useCampaignImportPanelLoad>;
