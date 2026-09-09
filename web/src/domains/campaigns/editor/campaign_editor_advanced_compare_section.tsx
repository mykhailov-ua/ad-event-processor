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
    <section >
      <h2 >Compare campaigns</h2>
      <div >
        <Label htmlFor="campaign-diff-against-id">Against campaign ID</Label>
        <Input
          id="campaign-diff-against-id"
          value={diffAgainstId}
          disabled={comparingDiff || fetching}
          placeholder="Other campaign UUID"
          onChange={(event) => onDiffAgainstIdChange(event.target.value)}
        />
        <p >
          Compare this campaign ({campaign.id}) against another campaign in the same customer.
        </p>
      </div>

      <div >
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
        <div >
          {diffResult.truncated ? (
            <Badge variant="outline">Diff truncated - showing first rows only</Badge>
          ) : null}
          {diffResult.rows.length === 0 ? (
            <p >No differences found.</p>
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
                    <TableCell >{row.label}</TableCell>
                    <TableCell >{formatReadonly(row.left_display)}</TableCell>
                    <TableCell >{formatReadonly(row.right_display)}</TableCell>
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
