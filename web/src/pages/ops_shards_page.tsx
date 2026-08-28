import { useCallback, useState } from 'react';
import { buildShardsUrl, shardCatchup, type ShardStatus } from '../helpers/ops_api.js';
import { ConfirmCancelledError } from '../helpers/confirmed_api.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { useResource } from '../helpers/use_resource.js';
import { OpsShardsPanel } from '../ui/ops/ops_shards_panel.js';

export function OpsShardsPage() {
  const { data, loading, error, reload } = useResource<ShardStatus>(buildShardsUrl());
  const [catchupBusy, setCatchupBusy] = useState(false);

  const onCatchup = useCallback(() => {
    setCatchupBusy(true);
    void (async () => {
      try {
        await shardCatchup();
        pushToastMessage({ title: 'Shard 0 catch-up', message: 'Completed' });
        reload();
      } catch (err) {
        if (err instanceof ConfirmCancelledError) return;
        pushToastMessage({
          title: 'Catch-up failed',
          message: err instanceof Error ? err.message : 'Catch-up failed',
        });
      } finally {
        setCatchupBusy(false);
      }
    })();
  }, [reload]);

  return (
    <OpsShardsPanel
      data={data}
      loading={loading}
      error={error}
      catchupBusy={catchupBusy}
      onCatchup={onCatchup}
    />
  );
}
