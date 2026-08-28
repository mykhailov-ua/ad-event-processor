import type { InvoiceDetail, InvoiceDetailTab } from '../../helpers/billing_api.js';
import { INVOICE_DETAIL_TABS } from '../../helpers/billing_api.js';
import { ContextBar } from '../shell/context_bar.js';
import { PageChrome } from '../system/page_chrome.js';
import { TabBar } from '../system/tab_bar.js';
import {
  InvoiceDeliveriesPanel,
  InvoiceHeaderPanel,
  InvoiceLedgerPanel,
  InvoicePdfPanel,
  InvoiceToolbar,
} from './invoice_tab_panels.js';
import styles from './invoice_detail.module.css';

export type InvoiceDetailProps = {
  invoiceId: string;
  invoice: InvoiceDetail;
  tab: InvoiceDetailTab;
  onTabChange: (tab: InvoiceDetailTab) => void;
  onReload: () => void;
};

function shortId(id: string): string {
  return id.length > 12 ? `${id.slice(0, 8)}...` : id;
}

export function InvoiceDetailView({
  invoiceId,
  invoice,
  tab,
  onTabChange,
  onReload,
}: InvoiceDetailProps) {
  return (
    <div className={styles.root}>
      <ContextBar parentLabel="Billing" parentTo="/billing" currentLabel={shortId(invoiceId)} />
      <PageChrome
        title={`Invoice ${invoice.billing_month ?? ''}`.trim() || 'Invoice'}
        badge={<span className={styles.mono}>{shortId(invoiceId)}</span>}
      />
      <InvoiceToolbar invoiceId={invoiceId} onReload={onReload} />
      <TabBar
        tabs={INVOICE_DETAIL_TABS}
        active={tab}
        onChange={(next) => onTabChange(next as InvoiceDetailTab)}
      />
      <div className={styles.panel} role="tabpanel">
        {tab === 'header' ? <InvoiceHeaderPanel invoice={invoice} /> : null}
        {tab === 'lines' ? <InvoiceLedgerPanel invoiceId={invoiceId} /> : null}
        {tab === 'deliveries' ? <InvoiceDeliveriesPanel invoiceId={invoiceId} /> : null}
        {tab === 'pdf' ? <InvoicePdfPanel invoiceId={invoiceId} /> : null}
      </div>
    </div>
  );
}
