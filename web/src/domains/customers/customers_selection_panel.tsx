import { Link } from 'react-router-dom';

import type { Customer } from '@/api/types';
import { PrimaryActionButton, SecondaryActionButton } from '@/shell/action_buttons';
import { ControlPlaneSelectionPanel } from '@/shell/control_plane_selection_panel';

export type CustomersSelectionPanelProps = {
  selectedCustomer: Customer | undefined;
  onClearSelection: () => void;
};

export function CustomersSelectionPanel({
  selectedCustomer,
  onClearSelection,
}: CustomersSelectionPanelProps) {
  const selectedTitle = selectedCustomer?.name ?? selectedCustomer?.id;

  return (
    <ControlPlaneSelectionPanel
      emptyHint="Select a customer in the list to open details."
      meta={
        selectedCustomer ? (
          <>
            {selectedCustomer.balance != null ? `Balance: ${selectedCustomer.balance}` : null}
            {selectedCustomer.currency ? ` / ${selectedCustomer.currency}` : null}
          </>
        ) : null
      }
      selectedTitle={selectedTitle}
      title="Customer actions"
    >
      {selectedCustomer?.id ? (
        <PrimaryActionButton asChild>
          <Link to={`/customers/${selectedCustomer.id}`}>Open detail</Link>
        </PrimaryActionButton>
      ) : null}
      <SecondaryActionButton type="button" onClick={onClearSelection}>
        Clear selection
      </SecondaryActionButton>
    </ControlPlaneSelectionPanel>
  );
}
