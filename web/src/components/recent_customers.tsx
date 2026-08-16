import * as storage from '../helpers/storage.js';
import { shortCustomerId } from '../helpers/customer_context.js';

export type RecentCustomersProps = {
  tenant?: boolean;
};

/**
 * Recent customer quick links from navigation storage.
 */
export function RecentCustomers({ tenant }: RecentCustomersProps) {
  if (tenant) return null;
  const ids = storage.getRecentCustomerIds();
  if (ids.length === 0) return null;
  return (
    <div className="recent-bar">
      <span className="recent-bar__label">Recent</span>
      {ids.map((id) => (
        <a key={id} href={`/customers/${id}`} className="recent-chip" title={id}>
          {shortCustomerId(id)}
        </a>
      ))}
    </div>
  );
}
