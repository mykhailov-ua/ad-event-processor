import { IntegrationsDebugger } from '@/domains/integrations/integrations_debugger';
import { useIntegrationsDebuggerPageWorkspace } from '@/domains/integrations/use_integrations_debugger_page_workspace';

export function IntegrationsDebuggerPage() {
  return <IntegrationsDebugger {...useIntegrationsDebuggerPageWorkspace()} />;
}
