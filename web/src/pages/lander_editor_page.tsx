import { useParams } from 'react-router-dom';
import { useResource } from '../helpers/use_resource.js';
import { LanderEditorDetail } from '../ui/landers/lander_editor_detail.js';
import { ErrorBlock } from '../ui/system/error_block.js';

export function LanderEditorPage() {
  const { id } = useParams();
  const landerId = id ?? '';

  const { reload } = useResource(landerId ? `/api/v1/landers/${encodeURIComponent(landerId)}/hosted-editor` : null, {
    skip: !landerId,
  });

  if (!landerId) {
    return <ErrorBlock error={new Error('missing lander id')} fallbackTitle="Invalid route" />;
  }

  return <LanderEditorDetail landerId={landerId} onReload={reload} />;
}
