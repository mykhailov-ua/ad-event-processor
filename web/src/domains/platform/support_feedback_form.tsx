import { PageChrome } from '@/components/system/page_chrome';
import { ErrorBlock } from '@/components/system/error_block';
import { PageSkeleton } from '@/components/system/page_skeleton';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import type { SupportFeedbackMeta } from '@/api/types';
import { COLD_PATH_MAX_BODY_CHARS } from '@/lib/body_limits';

export type SupportFeedbackFormProps = {
  meta: SupportFeedbackMeta | undefined;
  draftType: string;
  draftContactEmail: string;
  draftMessage: string;
  attachBundle: boolean;
  fetchingMeta: boolean;
  submitting: boolean;
  metaError: Error | undefined;
  submitError: Error | undefined;
  submittedId: string | undefined;
  onDraftTypeChange: (value: string) => void;
  onDraftContactEmailChange: (value: string) => void;
  onDraftMessageChange: (value: string) => void;
  onAttachBundleChange: (value: boolean) => void;
  onSubmit: () => void;
};

export function SupportFeedbackForm({
  meta,
  draftType,
  draftContactEmail,
  draftMessage,
  attachBundle,
  fetchingMeta,
  submitting,
  metaError,
  submitError,
  submittedId,
  onDraftTypeChange,
  onDraftContactEmailChange,
  onDraftMessageChange,
  onAttachBundleChange,
  onSubmit,
}: SupportFeedbackFormProps) {
  if (fetchingMeta && !meta && !metaError) {
    return <PageSkeleton />;
  }

  return (
    <PageChrome title="Support feedback">
      {meta ? (
        <dl className="grid gap-1 text-sm text-muted-foreground sm:grid-cols-2">
          {meta.deployment_id ? (
            <div>
              <dt>Deployment</dt>
              <dd className="font-mono text-xs text-foreground">{meta.deployment_id}</dd>
            </div>
          ) : null}
          {meta.binary_version ? (
            <div>
              <dt>Binary version</dt>
              <dd>{meta.binary_version}</dd>
            </div>
          ) : null}
        </dl>
      ) : null}

      {metaError ? <ErrorBlock title="Could not load feedback metadata" message={metaError.message} /> : null}

      <section className="ui-filter-panel max-w-xl">
        <div className="grid gap-2">
          <Label htmlFor="feedback-type">Type</Label>
          <Input
            id="feedback-type"
            value={draftType}
            onChange={(event) => onDraftTypeChange(event.target.value)}
          />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="feedback-email">Contact email (optional)</Label>
          <Input
            id="feedback-email"
            type="email"
            value={draftContactEmail}
            onChange={(event) => onDraftContactEmailChange(event.target.value)}
          />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="feedback-message">Message</Label>
          <Textarea
            id="feedback-message"
            maxLength={COLD_PATH_MAX_BODY_CHARS}
            value={draftMessage}
            onChange={(event) => onDraftMessageChange(event.target.value)}
          />
        </div>
        <div className="flex items-center gap-2">
          <Checkbox
            checked={attachBundle}
            id="feedback-attach-bundle"
            onCheckedChange={(checked) => onAttachBundleChange(checked === true)}
          />
          <Label className="font-normal" htmlFor="feedback-attach-bundle">
            Attach diagnostic bundle
          </Label>
        </div>
        <Button
          disabled={submitting || !draftType.trim() || !draftMessage.trim()}
          onClick={onSubmit}
          type="button"
        >
          Submit feedback
        </Button>
        {submittedId ? (
          <p className="text-sm text-muted-foreground">Feedback recorded: {submittedId}</p>
        ) : null}
        {submitError ? <ErrorBlock title="Submit failed" message={submitError.message} /> : null}
      </section>
    </PageChrome>
  );
}
