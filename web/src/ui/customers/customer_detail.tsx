import type { Customer, CustomerDetailTab } from '../../helpers/customers_api.js';
import { CUSTOMER_DETAIL_TABS } from '../../helpers/customers_api.js';
import { ContextBar } from '../shell/context_bar.js';
import { PageChrome } from '../system/page_chrome.js';
import { TabBar } from '../system/tab_bar.js';
import {
  CustomerApiKeysPanel,
  CustomerBalancePanel,
  CustomerCampaignsPanel,
  CustomerForecastPanel,
  CustomerKpiStrip,
  CustomerLedgerPanel,
  CustomerOverviewPanel,
  CustomerPaymentsPanel,
  CustomerStatementPanel,
  CustomerTaxPanel,
  CustomerToolbar,
  CustomerWalletPanel,
} from './customer_tab_panels.js';
import styles from './customer_detail.module.css';

export type CustomerDetailProps = {
  customerId: string;
  customer: Customer;
  tab: CustomerDetailTab;
  onTabChange: (tab: CustomerDetailTab) => void;
};

function shortId(id: string): string {
  return id.length > 12 ? `${id.slice(0, 8)}...` : id;
}

export function CustomerDetail({ customerId, customer, tab, onTabChange }: CustomerDetailProps) {
  return (
    <div className={styles.root}>
      <ContextBar parentLabel="Customers" parentTo="/customers" currentLabel={customer.name ?? customerId} />
      <PageChrome
        title={customer.name ?? 'Customer'}
        badge={<span className={styles.mono}>{shortId(customerId)}</span>}
      />
      <CustomerToolbar customerId={customerId} />
      {tab === 'overview' ? <CustomerKpiStrip customer={customer} /> : null}
      <TabBar
        tabs={CUSTOMER_DETAIL_TABS}
        active={tab}
        onChange={(next) => onTabChange(next as CustomerDetailTab)}
      />
      <div className={styles.panel} role="tabpanel">
        {tab === 'overview' ? <CustomerOverviewPanel customer={customer} /> : null}
        {tab === 'balance' ? <CustomerBalancePanel customerId={customerId} /> : null}
        {tab === 'ledger' ? <CustomerLedgerPanel customerId={customerId} /> : null}
        {tab === 'statement' ? <CustomerStatementPanel customerId={customerId} /> : null}
        {tab === 'forecast' ? <CustomerForecastPanel customerId={customerId} /> : null}
        {tab === 'wallet' ? <CustomerWalletPanel customerId={customerId} /> : null}
        {tab === 'tax' ? <CustomerTaxPanel customerId={customerId} /> : null}
        {tab === 'payments' ? <CustomerPaymentsPanel customerId={customerId} /> : null}
        {tab === 'campaigns' ? <CustomerCampaignsPanel customerId={customerId} /> : null}
        {tab === 'api_keys' ? <CustomerApiKeysPanel /> : null}
      </div>
    </div>
  );
}
