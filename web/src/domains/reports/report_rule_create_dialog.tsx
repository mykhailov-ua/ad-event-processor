import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import type { SourceQualityRow } from '@/api/types';
import { useReportRuleCreateDialogWorkspace } from '@/domains/reports/use_report_rule_create_dialog_workspace';
import type { ReportRuleFilterContext } from '@/lib/report_rule_snapshot';
import { PrimaryActionButton, SecondaryActionButton } from '@/shell/action_buttons';
import { ErrorBlock } from '@/shell/error_block';
import { adminKit, adminTypography } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';

export type ReportRuleCreateDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  reportKey: string;
  row: SourceQualityRow | undefined;
  filterContext: ReportRuleFilterContext;
  onCreated?: () => void;
};

export function ReportRuleCreateDialog({
  open,
  onOpenChange,
  reportKey,
  row,
  filterContext,
  onCreated,
}: ReportRuleCreateDialogProps) {
  const {
    action,
    onActionChange,
    name,
    onNameChange,
    threshold,
    onThresholdChange,
    creating,
    createError,
    canSubmit,
    campaignId,
    onCreate,
  } = useReportRuleCreateDialogWorkspace({
    open,
    reportKey,
    row,
    filterContext,
    onCreated: () => {
      onCreated?.();
      onOpenChange(false);
    },
  });

  return (
    <Dialog onOpenChange={onOpenChange} open={open}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Create automation rule</DialogTitle>
          <DialogDescription>
            Saves the current report filters and row keys as an automation rule snapshot. Rules run
            on the worker schedule (pause campaign or blacklist placement).
          </DialogDescription>
        </DialogHeader>
        <DialogBody className="grid gap-4">
          <div className={cn('grid', adminKit.fieldLabelGap)}>
            <Label htmlFor="report-rule-name">Rule name</Label>
            <Input
              disabled={creating}
              id="report-rule-name"
              value={name}
              onChange={(event) => onNameChange(event.target.value)}
            />
          </div>
          <div className={cn('grid', adminKit.fieldLabelGap)}>
            <Label htmlFor="report-rule-action">Action</Label>
            <Select
              disabled={creating}
              value={action}
              onValueChange={(value) => onActionChange(value as typeof action)}
            >
              <SelectTrigger id="report-rule-action">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="blacklist_placement">Blacklist placement</SelectItem>
                <SelectItem value="pause_campaign">Pause campaign</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className={cn('grid', adminKit.fieldLabelGap)}>
            <Label htmlFor="report-rule-threshold">Threshold (optional)</Label>
            <Input
              disabled={creating}
              id="report-rule-threshold"
              inputMode="decimal"
              placeholder="Leave empty for report defaults"
              value={threshold}
              onChange={(event) => onThresholdChange(event.target.value)}
            />
          </div>
          <p className={cn('m-0', adminTypography.bodyMuted)}>
            Campaign: {campaignId ?? 'missing - set campaign filter or pick a campaign row'}
          </p>
          {createError ? <ErrorBlock error={createError} title="Create rule failed" /> : null}
        </DialogBody>
        <DialogFooter>
          <SecondaryActionButton
            disabled={creating}
            onClick={() => onOpenChange(false)}
            type="button"
          >
            Cancel
          </SecondaryActionButton>
          <PrimaryActionButton
            disabled={!canSubmit}
            loading={creating}
            onClick={onCreate}
            type="button"
          >
            Create rule
          </PrimaryActionButton>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
