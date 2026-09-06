import { OpsOutbox } from '@/domains/ops/ops_outbox';
import { useOpsOutboxPageWorkspace } from '@/domains/ops/use_ops_outbox_page_workspace';

export function OpsOutboxPage() {
  return <OpsOutbox {...useOpsOutboxPageWorkspace()} />;
}
