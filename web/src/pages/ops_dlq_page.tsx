import { OpsDlqInbox } from '@/domains/ops/ops_dlq_inbox';
import { useOpsDlqPageWorkspace } from '@/domains/ops/use_ops_dlq_page_workspace';

export function OpsDlqPage() {
  return <OpsDlqInbox {...useOpsDlqPageWorkspace()} />;
}
