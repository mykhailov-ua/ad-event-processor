import type { SupportFeedbackPageWorkspace } from '@/domains/support/use_support_feedback_page_workspace';
import { adminSpacing } from '@/lib/admin_spacing';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { ErrorBlock } from '@/shell/error_block';
import { PageChrome } from '@/shell/page_chrome';
import {
  DirectoryFilterForm,
  FilterField,
  INLINE_FILTER_ACTION_GRID_CLASS,
} from '@/shell/filter_panel';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';

export type SupportFeedbackFormProps = SupportFeedbackPageWorkspace;

export function SupportFeedbackForm(workspace: SupportFeedbackFormProps) {
  const {
    meta,
    metaError,
    feedbackType,
    setFeedbackType,
    contactEmail,
    setContactEmail,
    message,
    setMessage,
    attachBundle,
    setAttachBundle,
    submitting,
    submitError,
    submittedId,
    onSubmit,
  } = workspace;

  const canSubmit = contactEmail.trim().length > 0 && message.trim().length > 0;

  return (
    <PageChrome
      title="Support feedback"
      description="Submit operator feedback to the platform team. Secondary surface; deep-link only."
    >
      {metaError ? <ErrorBlock error={metaError} title="Could not load feedback metadata" /> : null}
      {meta?.deployment_id || meta?.binary_version ? (
        <p>
          Deployment: {meta.deployment_id ?? '-'} | Binary: {meta.binary_version ?? '-'}
        </p>
      ) : null}

      <DirectoryFilterForm layout="auto-fill" onSubmit={(event) => event.preventDefault()}>
        <FilterField htmlFor="support-feedback-type" label="Type">
          <Select
            value={feedbackType}
            onValueChange={(value) => setFeedbackType(value as typeof feedbackType)}
          >
            <SelectTrigger id="support-feedback-type">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="bug">Bug</SelectItem>
              <SelectItem value="feature">Feature</SelectItem>
              <SelectItem value="support">Support</SelectItem>
            </SelectContent>
          </Select>
        </FilterField>
        <FilterField htmlFor="support-feedback-email" label="Contact email">
          <Input
            id="support-feedback-email"
            type="email"
            value={contactEmail}
            onChange={(event) => setContactEmail(event.target.value)}
          />
        </FilterField>
        <FilterField htmlFor="support-feedback-message" label="Message">
          <Textarea
            id="support-feedback-message"
            rows={6}
            value={message}
            onChange={(event) => setMessage(event.target.value)}
          />
        </FilterField>
        <div className={`flex items-center ${adminSpacing.gap.md}`}>
          <Checkbox
            checked={attachBundle}
            id="support-feedback-attach-bundle"
            onCheckedChange={(checked) => setAttachBundle(checked === true)}
          />
          <Label htmlFor="support-feedback-attach-bundle">Attach support bundle</Label>
        </div>
        <div className={INLINE_FILTER_ACTION_GRID_CLASS}>
          <Button disabled={submitting || !canSubmit} onClick={onSubmit} type="button">
            {submitting ? 'Submitting...' : 'Submit feedback'}
          </Button>
        </div>
      </DirectoryFilterForm>

      {submitError ? <ErrorBlock error={submitError} title="Could not submit feedback" /> : null}
      {submittedId ? <p role="status">Feedback recorded. Reference ID: {submittedId}</p> : null}
    </PageChrome>
  );
}
