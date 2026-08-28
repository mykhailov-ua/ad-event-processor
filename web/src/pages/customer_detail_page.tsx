import { useCallback } from 'react';
import { useParams, useSearchParams } from 'react-router-dom';
import * as auth from '../helpers/auth.js';
import { ApiError } from '../helpers/api_client.js';
import {
  parseCustomerDetailTab,
  type Customer,
  type CustomerDetailTab,
} from '../helpers/customers_api.js';
import { isTenantUser } from '../helpers/permissions.js';
import { useResource } from '../helpers/use_resource.js';
import { touchCustomerContext } from '../helpers/customer_context.js';
import { CustomerDetail } from '../ui/customers/customer_detail.js';
import { ErrorBlock } from '../ui/system/error_block.js';
import { PageSkeleton } from '../ui/system/page_skeleton.js';

export function CustomerDetailPage() {
  const { id } = useParams();
  const [searchParams, setSearchParams] = useSearchParams();
  const user = auth.getUser();
  const customerId = id ?? '';

  const tab = parseCustomerDetailTab(searchParams.get('tab'));

  const tenantBlocked =
    isTenantUser(user?.role) &&
    Boolean(user?.customer_id) &&
    user!.customer_id !== customerId;

  const listUrl = customerId ? `/api/v1/customers/${encodeURIComponent(customerId)}` : null;
  const { data, loading, error } = useResource<Customer>(tenantBlocked ? null : listUrl, {
    skip: !customerId || tenantBlocked,
  });

  const onTabChange = useCallback(
    (next: CustomerDetailTab) => {
      const params = new URLSearchParams(searchParams);
      if (next === 'overview') {
        params.delete('tab');
      } else {
        params.set('tab', next);
      }
      setSearchParams(params, { replace: true });
    },
    [searchParams, setSearchParams]
  );

  if (!customerId) {
    return <ErrorBlock error={new Error('missing customer id')} fallbackTitle="Invalid route" />;
  }

  if (tenantBlocked) {
    return (
      <ErrorBlock
        error={new ApiError(403, 'FORBIDDEN', 'Access denied')}
        fallbackTitle="Access denied"
      />
    );
  }

  if (loading && !data) {
    return <PageSkeleton rows={6} />;
  }

  if (error) {
    return <ErrorBlock error={error} fallbackTitle="Failed to load customer" />;
  }

  if (!data) {
    return <ErrorBlock error={new Error('empty customer')} fallbackTitle="Customer not found" />;
  }

  touchCustomerContext(customerId);

  return (
    <CustomerDetail
      customerId={customerId}
      customer={data}
      tab={tab}
      onTabChange={onTabChange}
    />
  );
}
