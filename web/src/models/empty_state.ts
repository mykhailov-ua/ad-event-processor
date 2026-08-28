export type BuyerEmptyKind =
  | 'campaigns_empty'
  | 'campaigns_blocked'
  | 'reports_empty'
  | 'forbidden'
  | 'session_customer';

export type BuyerEmptyCopy = {
  title: string;
  description: string;
  actionLabel?: string;
  actionHref?: string;
};

export function buyerEmptyCopy(kind: BuyerEmptyKind | string): BuyerEmptyCopy {
  switch (kind) {
    case 'campaigns_empty':
      return {
        title: 'No campaigns in this view',
        description: 'Change the status filter or confirm delivery has started for this customer.',
        actionLabel: 'Check placements report',
        actionHref: '/reports/placements',
      };
    case 'campaigns_blocked':
      return {
        title: 'Campaign list unavailable',
        description: 'Your account is missing a customer binding. Contact the platform operator.',
      };
    case 'reports_empty':
      return {
        title: 'No report rows for this range',
        description: 'Widen the date range or verify traffic is reaching tracked placements.',
        actionLabel: 'Review campaigns',
        actionHref: '/campaigns',
      };
    case 'forbidden':
      return {
        title: 'This page is outside your buyer scope',
        description: 'Use Campaigns, Reports, or Overview from the navigation menu.',
        actionLabel: 'Open campaigns',
        actionHref: '/campaigns',
      };
    case 'session_customer':
      return {
        title: 'Customer context required',
        description:
          'Sign in again or ask the operator to attach a customer to your buyer account.',
      };
    default:
      return {
        title: 'Nothing to show',
        description: 'Adjust filters or pick another task from Overview.',
        actionLabel: 'Back to overview',
        actionHref: '/',
      };
  }
}
