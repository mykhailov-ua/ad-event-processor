import type { Campaign, CampaignDiffResponse } from '@/api/types';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  DirectoryTable,
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from '@/shell/directory_table';
import { adminTypography } from '@/lib/admin_kit';
import {
  campaignEditorActionsRowClass,
  diffSeverityVariant,
  editorApiErrorBlock,
  formatReadonly,
} from '@/domains/campaigns/editor/campaign_editor_shared';

type CampaignEditorAdvancedCompareSectionProps = {
  campaign: Campaign;
  diffAgainstId: string;
  onDiffAgainstIdChange: (value: string) => void;
  comparingDiff: boolean;
  fetching: boolean;
  diffResult: CampaignDiffResponse | undefined;
  diffError: Error | undefined;
  onCompareDiff: () => void;
};

export function CampaignEditorAdvancedCompareSection({
  campaign,
  diffAgainstId,
  onDiffAgainstIdChange,
  comparingDiff,
  fetching,
  diffResult,
  diffError,
  onCompareDiff,
}: CampaignEditorAdvancedCompareSectionProps) {
  return (
    <section className="flex flex-col gap-4">
      <h2 className={adminTypography.sectionTitle}>Compare campaigns</h2>
      <div className="grid gap-2">
        <Label htmlFor="campaign-diff-against-id">Against campaign ID</Label>
        <Input
          id="campaign-diff-against-id"
          value={diffAgainstId}
          disabled={comparingDiff || fetching}
          placeholder="Other campaign UUID"
          onChange={(event) => onDiffAgainstIdChange(event.target.value)}
        />
        <p className={adminTypography.captionPlain}>
          Compare this campaign ({campaign.id}) against another campaign in the same customer.
        </p>
      </div>

      <div className={campaignEditorActionsRowClass}>
        <Button
          type="button"
          variant="secondary"
          disabled={comparingDiff || fetching}
          onClick={onCompareDiff}
        >
          {comparingDiff ? 'Comparing...' : 'Compare'}
        </Button>
      </div>

      {diffError
        ? editorApiErrorBlock(diffError, 'Campaign diff unavailable', 'Could not compare campaigns')
        : null}

      {diffResult ? (
        <div className="grid gap-3">
          {diffResult.truncated ? (
            <Badge variant="outline">Diff truncated - showing first rows only</Badge>
          ) : null}
          {diffResult.rows.length === 0 ? (
            <p className={adminTypography.bodyMuted}>No differences found.</p>
          ) : (
            <DirectoryTable>
              <TableHeader>
                <TableRow>
                  <DirectoryTableHead>Field</DirectoryTableHead>
                  <DirectoryTableHead>This campaign</DirectoryTableHead>
                  <DirectoryTableHead>Against campaign</DirectoryTableHead>
                  <DirectoryTableHead>Severity</DirectoryTableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {diffResult.rows.map((row) => (
                  <TableRow key={row.path}>
                    <TableCell className="font-medium">{row.label}</TableCell>
                    <TableCell className={adminTypography.monoData}>
                      {formatReadonly(row.left_display)}
                    </TableCell>
                    <TableCell className={adminTypography.monoData}>
                      {formatReadonly(row.right_display)}
                    </TableCell>
                    <TableCell>
                      <Badge variant={diffSeverityVariant(row.severity)}>{row.severity}</Badge>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </DirectoryTable>
          )}
        </div>
      ) : null}
    </section>
  );
}
