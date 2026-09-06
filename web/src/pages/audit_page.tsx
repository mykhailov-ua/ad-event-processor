import { AuditDirectory } from '@/domains/audit/audit_directory';
import { useAuditPageWorkspace } from '@/domains/audit/use_audit_page_workspace';

export function AuditPage() {
  return <AuditDirectory {...useAuditPageWorkspace()} />;
}
