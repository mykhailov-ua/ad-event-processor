import { LanderHostedEditor } from '@/domains/creative/lander_hosted_editor';
import { useLanderEditorPageWorkspace } from '@/domains/creative/use_lander_editor_page_workspace';

export function LanderEditorPage() {
  return <LanderHostedEditor {...useLanderEditorPageWorkspace()} />;
}
