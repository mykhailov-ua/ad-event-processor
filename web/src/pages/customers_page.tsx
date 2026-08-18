import { useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import * as auth from '../helpers/auth.js';
import { isTenantUser } from '../helpers/permissions.js';
import { touchCustomerContext } from '../helpers/customer_context.js';
import { formatUsdDecimal } from '../helpers/money.js';
import type { CustomerDTO, CustomerListResponse } from '../types/customer.js';
import { createSortState, sortRows, toggleSort } from '../lib/table_sort.js';
import { useResource } from '../helpers/use_resource.js';
import { ErrorBlock } from '../components/error_block.js';
import { FilterToolbar } from '../components/filter_toolbar.js';
import { Icon } from '../components/icon.js';
import { PaginationBar } from '../components/pagination_bar.js';
import { RecentCustomers } from '../components/recent_customers.js';

const PAGE_SIZE = 50;

function buildUrl(page: number) {
  const offset = page * PAGE_SIZE;
  return `/api/v1/customers?limit=${PAGE_SIZE}&offset=${offset}`;
}

function formatBalance(bal: unknown) {
  if (!bal) return '—';
  return formatUsdDecimal(String(bal));
}

function TableSkeleton({ cols, rows = 5 }: { cols: number; rows?: number }) {
  return (
    <>
      {Array.from({ length: rows }, (_, rowIndex) => (
        <tr key={`skel-${rowIndex}`} className="data-table__row--skeleton" aria-hidden="true">
          {Array.from({ length: cols }, (__, colIndex) => (
            <td key={`skel-${rowIndex}-${colIndex}`}>
              <span className="skeleton-bar" />
            </td>
          ))}
        </tr>
      ))}
    </>
  );
}

export function CustomersPage() {
  const navigate = useNavigate();
  const user = auth.getUser();
  const tenant = isTenantUser(user?.role);
  const tenantId = user?.customer_id;

  const [page, setPage] = useState(0);
  const [searchQuery, setSearchQuery] = useState('');
  const [sortState, setSortState] = useState(() => createSortState('name', 'asc'));

  useEffect(() => {
    if (tenant && tenantId) {
      navigate(`/customers/${tenantId}`, { replace: true });
    }
  }, [tenant, tenantId, navigate]);

  const { data, loading, error } = useResource<CustomerListResponse>(
    tenant && tenantId ? null : buildUrl(page),
    { skip: Boolean(tenant && tenantId) }
  );

  const customers = useMemo(() => {
    const sorted = sortRows(data?.items ?? [], sortState, {
      name: (c: CustomerDTO) => c.name ?? '',
      balance: (c: CustomerDTO) => Number(c.balance ?? 0),
      currency: (c: CustomerDTO) => c.currency ?? '',
      active_campaigns: (c: CustomerDTO) => Number(c.active_campaigns ?? 0),
      created_at: (c: CustomerDTO) => c.created_at ?? '',
    });
    const q = searchQuery.trim().toLowerCase();
    if (!q) return sorted;
    return sorted.filter(
      (c) => (c.name ?? '').toLowerCase().includes(q) || (c.id ?? '').toLowerCase().includes(q)
    );
  }, [data?.items, sortState, searchQuery]);

  if (tenant && tenantId) {
    return <span className="text-muted">Redirecting…</span>;
  }

  if (error) {
    return <ErrorBlock error={error} fallbackTitle="Failed to load customers" />;
  }

  const total = data?.total ?? 0;
  const totalPages = Math.ceil(total / PAGE_SIZE) || 1;

  const onSort = (key: string) => {
    setSortState((prev) => {
      const next = { ...prev };
      toggleSort(next, key);
      return next;
    });
  };

  const sortHeader = (label: string, key: string) => {
    const active = sortState.key === key;
    const iconName = active
      ? sortState.dir === 'asc'
        ? 'chevron-up'
        : 'chevron-down'
      : 'arrow-up-down';
    return (
      <th
        key={key}
        scope="col"
        className={['data-table__th--sortable', active ? 'data-table__th--sorted' : '']
          .filter(Boolean)
          .join(' ')}
        aria-sort={active ? (sortState.dir === 'asc' ? 'ascending' : 'descending') : 'none'}
        tabIndex={0}
        onClick={() => onSort(key)}
        onKeyDown={(e) => {
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault();
            onSort(key);
          }
        }}
      >
        <span className="data-table__th-label">{label}</span>
        <Icon name={iconName} size={13} className="data-table__sort-icon" />
      </th>
    );
  };

  return (
    <>
      <div className="page-header">
        <div className="page-header__row">
          <div className="flex items-center gap-2">
            <Icon name="users" size={20} className="text-muted" />
            <h1 className="page-header__title">Customers</h1>
          </div>
          <span className="text-muted text-sm">{loading ? '' : `${total} total`}</span>
        </div>
        <RecentCustomers tenant={tenant} />
      </div>

      <div className="mb-4">
        <FilterToolbar
          search
          searchPlaceholder="Filter by name or ID…"
          searchValue={searchQuery}
          onSearch={setSearchQuery}
          pagination={
            totalPages > 1 ? (
              <PaginationBar
                label={`${page + 1} / ${totalPages}`}
                prevDisabled={page === 0}
                nextDisabled={page >= totalPages - 1}
                onPrev={() => setPage((p) => Math.max(0, p - 1))}
                onNext={() => setPage((p) => p + 1)}
              />
            ) : null
          }
        />
      </div>

      <div className="table-wrapper table-wrapper--scroll elevation-raised">
        <table className="data-table">
          <thead>
            <tr>
              {sortHeader('Name', 'name')}
              {sortHeader('Balance', 'balance')}
              {sortHeader('Currency', 'currency')}
              {sortHeader('Active Campaigns', 'active_campaigns')}
              {sortHeader('Created', 'created_at')}
            </tr>
          </thead>
          <tbody>
            {loading && customers.length === 0 ? <TableSkeleton cols={5} /> : null}
            {!loading && customers.length === 0 ? (
              <tr>
                <td colSpan={5}>
                  <div className="empty-state">
                    <Icon
                      name="file-text"
                      size={28}
                      className="empty-state__icon text-muted mb-2"
                    />
                    <div className="empty-state__title">No customers found</div>
                    <div className="empty-state__desc text-muted text-sm">
                      Customers appear after they are created in the system.
                    </div>
                    <button
                      type="button"
                      className="btn btn--secondary btn--sm empty-state__action"
                      onClick={() => navigate('/billing')}
                    >
                      Open billing
                    </button>
                  </div>
                </td>
              </tr>
            ) : null}
            {customers.map((c) => (
              <tr
                key={c.id}
                id={`row-customer-${c.id}`}
                className="data-table__row--clickable"
                tabIndex={0}
                onClick={() => {
                  touchCustomerContext(c.id);
                  navigate(`/customers/${c.id}`);
                }}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' || e.key === ' ') {
                    e.preventDefault();
                    touchCustomerContext(c.id);
                    navigate(`/customers/${c.id}`);
                  }
                }}
              >
                <td className="font-medium">{c.name}</td>
                <td className="font-mono">{formatBalance(c.balance)}</td>
                <td>{c.currency ?? 'USD'}</td>
                <td>{String(c.active_campaigns ?? 0)}</td>
                <td className="text-muted">
                  {c.created_at ? new Date(c.created_at).toLocaleDateString() : '-'}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </>
  );
}
