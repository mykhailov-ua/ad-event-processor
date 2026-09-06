import { SupportFeedbackForm } from '@/domains/platform/support_feedback_form';
import { useSupportFeedbackPageWorkspace } from '@/domains/platform/use_support_feedback_page_workspace';

export function SupportFeedbackPage() {
  return <SupportFeedbackForm {...useSupportFeedbackPageWorkspace()} />;
}
