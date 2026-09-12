import { SupportFeedbackForm } from '@/domains/support/support_feedback_form';
import { useSupportFeedbackPageWorkspace } from '@/domains/support/use_support_feedback_page_workspace';

export function SupportFeedbackPage() {
  return <SupportFeedbackForm {...useSupportFeedbackPageWorkspace()} />;
}
