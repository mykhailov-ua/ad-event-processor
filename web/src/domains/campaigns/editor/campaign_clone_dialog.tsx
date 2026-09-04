import { useCallback, useEffect, useState } from 'react';
import { Link } from 'react-router-dom';

import {
  cloneCampaign,
  previewCampaignClone,
  type CloneCampaignOptions,
  type CloneCampaignPreview,
} from '@/api/campaigns_api';
import { isAbortError } from '@/api/client';
import { ErrorBlock } from '@/shell/error_block';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  buildCloneRequestBody,
  DEFAULT_CLONE_OPTIONS,
} from '@/domains/campaigns/editor/campaign_clone_request';

export type CampaignCloneDialogProps = {
  campaignId: string | undefined;
  campaignName?: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCloned?: (newCampaignId: string) => void;
};

const CLONE_OPTION_FIELDS: { field: keyof CloneCampaignOptions; label: string }[] = [
  { field: 'include_flow', label: 'Include flow' },
  { field: 'include_postbacks', label: 'Include postbacks' },
  { field: 'include_fraud', label: 'Include fraud config' },
  { field: 'include_placement_blocks', label: 'Include placement blocks' },
  { field: 'reset_spend', label: 'Reset spend counters' },
];

export function CampaignCloneDialog({
  campaignId,
  campaignName,
  open,
  onOpenChange,
  onCloned,
}: CampaignCloneDialogProps) {
  const [nameSuffix, setNameSuffix] = useState(' (copy)');
  const [cloneOptions, setCloneOptions] = useState<CloneCampaignOptions>(DEFAULT_CLONE_OPTIONS);
  const [preview, setPreview] = useState<CloneCampaignPreview | undefined>();
  const [previewError, setPreviewError] = useState<Error | undefined>();
  const [previewing, setPreviewing] = useState(false);
  const [cloning, setCloning] = useState(false);
  const [cloneError, setCloneError] = useState<Error | undefined>();
  const [clonedId, setClonedId] = useState<string | undefined>();

  useEffect(() => {
    if (!open) {
      setPreview(undefined);
      setPreviewError(undefined);
      setCloneError(undefined);
      setClonedId(undefined);
      setNameSuffix(' (copy)');
      setCloneOptions(DEFAULT_CLONE_OPTIONS);
    }
  }, [open]);

  const onPreview = useCallback(() => {
    if (!campaignId) {
      return;
    }
    setPreviewing(true);
    setPreviewError(undefined);
    void previewCampaignClone(campaignId, buildCloneRequestBody(nameSuffix, cloneOptions))
      .then((result) => setPreview(result))
      .catch((err: unknown) => {
        if (!isAbortError(err)) {
          setPreviewError(err instanceof Error ? err : new Error(String(err)));
        }
      })
      .finally(() => setPreviewing(false));
  }, [campaignId, cloneOptions, nameSuffix]);

  const onClone = useCallback(() => {
    if (!campaignId) {
      return;
    }
    setCloning(true);
    setCloneError(undefined);
    void cloneCampaign(campaignId, buildCloneRequestBody(nameSuffix, cloneOptions), {
      idempotencyKey: crypto.randomUUID(),
    })
      .then((result) => {
        setClonedId(result.id);
        onCloned?.(result.id);
      })
      .catch((err: unknown) => {
        if (!isAbortError(err)) {
          setCloneError(err instanceof Error ? err : new Error(String(err)));
        }
      })
      .finally(() => setCloning(false));
  }, [campaignId, cloneOptions, nameSuffix, onCloned]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Clone campaign</DialogTitle>
          <DialogDescription>
            {campaignName ? (
              <>
                Source: <strong>{campaignName}</strong>
              </>
            ) : (
              'Clone the selected campaign.'
            )}
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-3">
          <div className="grid gap-1">
            <Label htmlFor="clone-name-suffix">Name suffix</Label>
            <Input
              id="clone-name-suffix"
              value={nameSuffix}
              onChange={(event) => setNameSuffix(event.target.value)}
            />
          </div>

          <div className="flex flex-col gap-3">
            {CLONE_OPTION_FIELDS.map(({ field, label }) => (
              <label key={field} className="flex items-center gap-2">
                <Checkbox
                  checked={cloneOptions[field] ?? DEFAULT_CLONE_OPTIONS[field]}
                  onCheckedChange={(checked) =>
                    setCloneOptions((current) => ({ ...current, [field]: checked === true }))
                  }
                />
                <span>{label}</span>
              </label>
            ))}
          </div>

          {preview ? (
            <p className="text-sm text-muted-foreground">
              Preview name: <strong>{preview.name}</strong>
            </p>
          ) : null}
          {previewError ? <ErrorBlock message={previewError.message} title="Preview failed" /> : null}
          {cloneError ? <ErrorBlock message={cloneError.message} title="Clone failed" /> : null}
          {clonedId ? (
            <p>
              Created{' '}
              <Button asChild type="button" variant="link">
                <Link to={`/campaigns/${clonedId}/edit`}>{clonedId}</Link>
              </Button>
            </p>
          ) : null}
        </div>

        <DialogFooter className="gap-2">
          <Button disabled={!campaignId || previewing} type="button" variant="outline" onClick={onPreview}>
            {previewing ? 'Previewing...' : 'Preview'}
          </Button>
          <Button disabled={!campaignId || cloning || Boolean(clonedId)} type="button" onClick={onClone}>
            {cloning ? 'Cloning...' : 'Clone'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
