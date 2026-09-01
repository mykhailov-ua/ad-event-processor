import { FilterApplyButton } from '@/components/system/action_buttons';
import { PageChrome } from '@/components/system/page_chrome';
import { PageSkeleton } from '@/components/system/page_skeleton';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import type { RtbReconcileExport, RtbShadowDiffSnapshot } from '@/api/types';
import { JsonDashboardView } from '@/domains/dashboards/json_dashboard_view';
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
        className="grid max-w-xl grid-cols-[1fr_1fr_auto] items-end gap-4"
        onSubmit={(event) => {
          event.preventDefault();
          onApply();
        }}
      >
        <div className="grid gap-2">
          <Label htmlFor="rtb-shadow-window">Window</Label>
          <Input
            id="rtb-shadow-window"
            placeholder="1h"
            value={draftWindow}
            onChange={(event) => onDraftWindowChange(event.target.value)}
          />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="rtb-reconcile-request-id">Reconcile request_id</Label>
          <Input
            id="rtb-reconcile-request-id"
            value={draftRequestId}
            onChange={(event) => onDraftRequestIdChange(event.target.value)}
          />
        </div>
        <FilterApplyButton disabled={fetching}>Load</FilterApplyButton>
      </form>

      {shadow ? (
        <section className="grid gap-2">
          <h2 className="text-base font-semibold">Shadow diff</h2>
          <JsonDashboardView payload={shadow as unknown as Record<string, unknown>} />
        </section>
      ) : null}

      {reconcile ? (
        <section className="grid gap-2">
          <h2 className="text-base font-semibold">Reconcile export</h2>
          <JsonDashboardView payload={reconcile as unknown as Record<string, unknown>} />
        </section>
      ) : null}

      {error && hasSnapshot ? rtbPanelError(error, 'Refresh failed') : null}
    </PageChrome>
  );
}
