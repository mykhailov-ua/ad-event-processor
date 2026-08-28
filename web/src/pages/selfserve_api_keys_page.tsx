import { ApiKeysPanel } from '../ui/selfserve/api_keys_panel.js';
import { SelfServeShell } from '../ui/selfserve/selfserve_shell.js';

export function SelfServeApiKeysPage() {
  return (
    <SelfServeShell>
      <ApiKeysPanel />
    </SelfServeShell>
  );
}
