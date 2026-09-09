import { Button } from '@/components/ui/button';
import type {
  OpsConsentProofsResponse,
  OpsMlModelEvalResponse,
  OpsMlModelStatusResponse,
  OpsRumResponse,
} from '@/api/types';
import { JsonPayloadView } from '@/shell/json_payload_view';
import {
  OpsActionGroup,
  OpsPageWithLoad,
} from '@/domains/ops/ops_page_shell';

export type OpsRumProps = {
  payload: OpsRumResponse | undefined;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  onLoad: () => void;
};

export function OpsRum({ payload, fetching, error, hasSnapshot, onLoad }: OpsRumProps) {
  return (
    <OpsPageWithLoad
      blockingErrorTitle="Could not load RUM samples"
      fetchState={{ fetching, error, hasSnapshot }}
      title="RUM"
      actions={
        <OpsActionGroup label="RUM">
          <Button disabled={fetching} loading={fetching} type="button" onClick={onLoad}>
            Load samples
          </Button>
        </OpsActionGroup>
      }
    >
      {payload ? (
        <JsonPayloadView payload={payload} />
      ) : (
        <p >Load RUM samples from the control plane.</p>
      )}
    </OpsPageWithLoad>
  );
}
