import { useCallback, useMemo } from 'react';
import { useSearchParams } from 'react-router-dom';
import * as auth from '../helpers/auth.js';
import { can } from '../helpers/permissions.js';
import { buildBuyerDashboardUrl } from '../helpers/selfserve_api.js';
import { useResource } from '../helpers/use_resource.js';
import type { BuyerPortfolioResponse } from '../helpers/selfserve_api.js';
import { PortfolioPanel } from '../ui/selfserve/portfolio_panel.js';
import { SelfServeShell } from '../ui/selfserve/selfserve_shell.js';

function defaultRange(): { from: string; to: string } {
  const to = new Date();
  const from = new Date(to.getTime() - 7 * 24 * 60 * 60 * 1000);
  return { from: from.toISOString(), to: to.toISOString() };
}

export function SelfServeHomePage() {
  const user = auth.getUser();
  const customerId = user?.customer_id ?? '';
  const [searchParams] = useSearchParams();
  const range = useMemo(() => {
    const from = searchParams.get('from') ?? defaultRange().from;
    const to = searchParams.get('to') ?? defaultRange().to;
    return { from, to };
  }, [searchParams]);

  const url = buildBuyerDashboardUrl({
    customer_id: customerId || undefined,
    from: range.from,
    to: range.to,
  });

  const { data, loading, error, reload } = useResource<BuyerPortfolioResponse>(url);
  const permissions = user?.permissions ?? [];
  const canMutate =
    can(permissions, 'campaigns:write') || can(permissions, 'campaigns:pause');

  const onReload = useCallback(() => {
    reload();
  }, [reload]);

  return (
    <SelfServeShell>
      <PortfolioPanel
        data={data}
        loading={loading}
        error={error}
        canMutate={canMutate}
        onReload={onReload}
      />
    </SelfServeShell>
  );
}
