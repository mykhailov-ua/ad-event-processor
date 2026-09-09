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
    <section >
      <h2 >Macro preview</h2>
      <div >
        <div >
          <Label htmlFor="macro-preview-sub1">sub1</Label>
          <Input
            id="macro-preview-sub1"
            value={macroPreviewForm.sub1}
            disabled={macroPreviewing || fetching}
            onChange={(event) => onMacroPreviewFieldChange('sub1', event.target.value)}
          />
        </div>
        <div >
          <Label htmlFor="macro-preview-country">country</Label>
          <Input
            id="macro-preview-country"
            value={macroPreviewForm.country}
            disabled={macroPreviewing || fetching}
            onChange={(event) => onMacroPreviewFieldChange('country', event.target.value)}
          />
        </div>
        <div >
          <Label htmlFor="macro-preview-click-id">click_id</Label>
          <Input
            id="macro-preview-click-id"
            value={macroPreviewForm.click_id}
            disabled={macroPreviewing || fetching}
            onChange={(event) => onMacroPreviewFieldChange('click_id', event.target.value)}
          />
        </div>
      </div>

      <div >
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
        <div >
          <div >
            <p >Resolved click URL</p>
            <p >
              {formatReadonly(macroPreviewResult.resolved_click_url)}
            </p>
          </div>
          {macroPreviewResult.resolved_postback_url ? (
            <div >
              <p >Resolved postback URL</p>
              <p >
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
