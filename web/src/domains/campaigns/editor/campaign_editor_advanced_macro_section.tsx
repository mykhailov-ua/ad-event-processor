import { adminTypography } from '@/lib/admin_kit';
import type { MacroPreviewResponse } from '@/api/types';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  campaignEditorActionsRowClass,
  campaignEditorSectionClass,
  editorApiErrorBlock,
  formatReadonly,
  StringList,
} from '@/domains/campaigns/editor/campaign_editor_shared';
import { cn } from '@/lib/utils';

type CampaignEditorAdvancedMacroSectionProps = {
  macroPreviewForm: { sub1: string; country: string; click_id: string };
  onMacroPreviewFieldChange: (field: 'sub1' | 'country' | 'click_id', value: string) => void;
  macroPreviewing: boolean;
  fetching: boolean;
  macroPreviewResult: MacroPreviewResponse | undefined;
  macroPreviewError: Error | undefined;
  onMacroPreview: () => void;
};

export function CampaignEditorAdvancedMacroSection({
  macroPreviewForm,
  onMacroPreviewFieldChange,
  macroPreviewing,
  fetching,
  macroPreviewResult,
  macroPreviewError,
  onMacroPreview,
}: CampaignEditorAdvancedMacroSectionProps) {
  return (
    <section className="flex flex-col gap-4">
      <h2 className={adminTypography.sectionTitle}>Macro preview</h2>
      <div className="grid gap-2 sm:grid-cols-3">
        <div className="grid gap-2">
          <Label htmlFor="macro-preview-sub1">sub1</Label>
          <Input
            id="macro-preview-sub1"
            value={macroPreviewForm.sub1}
            disabled={macroPreviewing || fetching}
            onChange={(event) => onMacroPreviewFieldChange('sub1', event.target.value)}
          />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="macro-preview-country">country</Label>
          <Input
            id="macro-preview-country"
            value={macroPreviewForm.country}
            disabled={macroPreviewing || fetching}
            onChange={(event) => onMacroPreviewFieldChange('country', event.target.value)}
          />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="macro-preview-click-id">click_id</Label>
          <Input
            id="macro-preview-click-id"
            value={macroPreviewForm.click_id}
            disabled={macroPreviewing || fetching}
            onChange={(event) => onMacroPreviewFieldChange('click_id', event.target.value)}
          />
        </div>
      </div>

      <div className={campaignEditorActionsRowClass}>
        <Button
          type="button"
          variant="secondary"
          disabled={macroPreviewing || fetching}
          onClick={onMacroPreview}
        >
          {macroPreviewing ? 'Previewing...' : 'Preview'}
        </Button>
      </div>

      {macroPreviewError
        ? editorApiErrorBlock(
            macroPreviewError,
            'Macro preview unavailable',
            'Could not preview macros'
          )
        : null}

      {macroPreviewResult ? (
        <div className={cn(campaignEditorSectionClass, 'gap-3')}>
          <div className="grid gap-2">
            <p className={adminTypography.label}>Resolved click URL</p>
            <p className={cn('break-all', adminTypography.monoData, 'text-muted-foreground')}>
              {formatReadonly(macroPreviewResult.resolved_click_url)}
            </p>
          </div>
          {macroPreviewResult.resolved_postback_url ? (
            <div className="grid gap-2">
              <p className={adminTypography.label}>Resolved postback URL</p>
              <p className={cn('break-all', adminTypography.monoData, 'text-muted-foreground')}>
                {macroPreviewResult.resolved_postback_url}
              </p>
            </div>
          ) : null}
          <StringList title="Warnings" items={macroPreviewResult.warnings} />
          <StringList title="Unresolved macros" items={macroPreviewResult.unresolved_macros} />
        </div>
      ) : null}
    </section>
  );
}
