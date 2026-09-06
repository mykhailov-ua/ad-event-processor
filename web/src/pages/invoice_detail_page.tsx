import { InvoiceDetail } from '@/domains/billing/invoice_detail';
import { useInvoiceDetailPageWorkspace } from '@/domains/billing/use_invoice_detail_page_workspace';

export function InvoiceDetailPage() {
  return <InvoiceDetail {...useInvoiceDetailPageWorkspace()} />;
}
