import { Button } from '@/components/ui/button';
import {
  customerDetailTabs,
  type CustomerDetailTab,
} from '@/domains/customers/customer_detail_types';

export type CustomerDetailTabBarProps = {
  tab: CustomerDetailTab;
  paymentEnabled: boolean;
  onTabChange: (tab: CustomerDetailTab) => void;
};

export function CustomerDetailTabBar({
  tab,
  paymentEnabled,
  onTabChange,
}: CustomerDetailTabBarProps) {
  const tabs = customerDetailTabs(paymentEnabled);

  return (
    <div >
      {tabs.map((item) => (
        <Button
          key={item.id}
          aria-pressed={tab === item.id}
          type="button"
          variant={tab === item.id ? 'secondary' : 'ghost'}
          onClick={() => onTabChange(item.id)}
        >
          {item.label}
        </Button>
      ))}
    </div>
  );
}
