import { PageChrome } from '@/shell/page_chrome';
import { PageSkeleton } from '@/shell/page_skeleton';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import type { RtbFloorsApplyResult } from '@/api/types';
import { RtbNav, RtbLicenseStub, rtbPanelError } from '@/domains/rtb/rtb_nav';
import { displayMicro } from '@/lib/display';

export type RtbFloorsApplyPanelProps = {
  result: RtbFloorsApplyResult | undefined;
  draftPlacementIds: string;
  dryRun: boolean;
  applying: boolean;
  error: Error | undefined;
  licenseGated: boolean;
  onDraftPlacementIdsChange: (value: string) => void;
  onDryRunChange: (value: boolean) => void;
  onApply: () => void;
};

export function RtbFloorsApplyPanel({
  result,
  draftPlacementIds,
  dryRun,
  applying,
  error,
  licenseGated,
  onDraftPlacementIdsChange,
  onDryRunChange,
  onApply,
}: RtbFloorsApplyPanelProps) {
  if (licenseGated) {
    return (
      <PageChrome title="RTB floors">
        <RtbNav />
        <RtbLicenseStub />
      </PageChrome>
    );
  }

  return (
    <PageChrome title="RTB floors">
      <RtbNav />

      <section className="grid max-w-xl gap-4">
        <div className="grid gap-2">
          <Label htmlFor="rtb-floor-placements">placement_ids (one per line)</Label>
          <Textarea
            id="rtb-floor-placements"
            rows={4}
            value={draftPlacementIds}
            onChange={(event) => onDraftPlacementIdsChange(event.target.value)}
          />
        </div>
        <div className="flex items-center gap-2">
          <Checkbox
            checked={dryRun}
            id="rtb-floors-dry-run"
            onCheckedChange={(checked) => onDryRunChange(checked === true)}
          />
          <Label htmlFor="rtb-floors-dry-run">dry_run</Label>
        </div>
        <Button disabled={applying} onClick={onApply} type="button">
          Apply floors
        </Button>
      </section>

      {error ? rtbPanelError(error, 'Floors apply failed') : null}

      {result ? (
        <section className="grid gap-2">
          <div className="flex flex-wrap gap-2 text-sm">
            {result.dry_run ? <Badge variant="secondary">dry run</Badge> : null}
            <Badge variant="outline">applied {result.applied ?? 0}</Badge>
            <Badge variant="outline">outbox {result.outbox_rows ?? 0}</Badge>
          </div>
          {(result.suggestions ?? []).length > 0 ? (
            <div className="ui-table-frame">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Placement</TableHead>
                    <TableHead>Deal</TableHead>
                    <TableHead>Current floor</TableHead>
                    <TableHead>Suggested</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {(result.suggestions ?? []).map((row) => (
                    <TableRow key={`${row.placement_id}-${row.deal_id}`}>
                      <TableCell>{row.placement_id ?? ''}</TableCell>
                      <TableCell>{row.deal_id ?? ''}</TableCell>
                      <TableCell>{displayMicro(row.current_floor_micro)}</TableCell>
                      <TableCell>{displayMicro(row.suggested_floor_micro)}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          ) : null}
        </section>
      ) : applying ? (
        <PageSkeleton />
      ) : null}
    </PageChrome>
  );
}
