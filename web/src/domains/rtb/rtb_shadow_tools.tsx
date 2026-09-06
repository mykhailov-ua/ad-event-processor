import { FilterApplyButton } from '@/shell/action_buttons';
import { FilterField, INLINE_FILTER_ACTION_GRID_TWO_FIELDS_CLASS } from '@/shell/filter_panel';
import { PageChrome } from '@/shell/page_chrome';
import { PageSkeleton } from '@/shell/page_skeleton';
import { Input } from '@/components/ui/input';
import type { RtbReconcileExport, RtbShadowDiffSnapshot } from '@/api/types';
import { JsonPayloadView } from '@/shell/json_payload_view';
import { RtbNav, RtbLicenseStub, rtbPanelError } from '@/domains/rtb/rtb_nav';

export type RtbShadowToolsProps = {
  shadow: RtbShadowDiffSnapshot | undefined;
  reconcile: RtbReconcileExport | undefined;
  draftWindow: string;
  draftRequestId: string;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  licenseGated: boolean;
  onDraftWindowChange: (value: string) => void;
  onDraftRequestIdChange: (value: string) => void;
  onApply: () => void;
};

export function RtbShadowTools({
  shadow,
  reconcile,
  draftWindow,
  draftRequestId,
  fetching,
  error,
  hasSnapshot,
  licenseGated,
  onDraftWindowChange,
  onDraftRequestIdChange,
  onApply,
}: RtbShadowToolsProps) {
  if (licenseGated) {
    return (
      <PageChrome title="RTB shadow and reconcile">
        <RtbNav />
        <RtbLicenseStub />
      </PageChrome>
    );
  }

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return (
      <PageChrome title="RTB shadow and reconcile">
        <RtbNav />
        {rtbPanelError(error, 'Could not load RTB shadow data')}
      </PageChrome>
    );
  }

  return (
    <PageChrome title="RTB shadow and reconcile">
      <RtbNav />

      <form
        className={INLINE_FILTER_ACTION_GRID_TWO_FIELDS_CLASS}
        onSubmit={(event) => {
          event.preventDefault();
          onApply();
        }}
      >
        <FilterField htmlFor="rtb-shadow-window" label="Window">
          <Input
            id="rtb-shadow-window"
            placeholder="1h"
            value={draftWindow}
            onChange={(event) => onDraftWindowChange(event.target.value)}
          />
        </FilterField>
        <FilterField htmlFor="rtb-reconcile-request-id" label="Reconcile request_id">
          <Input
            id="rtb-reconcile-request-id"
            value={draftRequestId}
            onChange={(event) => onDraftRequestIdChange(event.target.value)}
          />
        </FilterField>
        <FilterApplyButton disabled={fetching}>Load</FilterApplyButton>
      </form>

      {shadow ? (
        <section className="grid gap-2">
          <h2 className="text-base font-semibold">Shadow diff</h2>
          <JsonPayloadView payload={shadow} />
        </section>
      ) : null}

      {reconcile ? (
        <section className="grid gap-2">
          <h2 className="text-base font-semibold">Reconcile export</h2>
          <JsonPayloadView payload={reconcile} />
        </section>
      ) : null}

      {error && hasSnapshot ? rtbPanelError(error, 'Refresh failed') : null}
    </PageChrome>
  );
}
