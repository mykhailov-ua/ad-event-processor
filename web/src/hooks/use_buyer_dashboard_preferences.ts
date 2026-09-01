import { useCallback, useState } from 'react';

import {
  type BuyerDashboardPreferences,
  loadBuyerDashboardPreferences,
  saveBuyerDashboardPreferences,
} from '@/domains/dashboards/dashboard_preferences';

export function useBuyerDashboardPreferences() {
  const [preferences, setPreferences] = useState<BuyerDashboardPreferences>(() =>
    loadBuyerDashboardPreferences(),
  );

  const applyPreferences = useCallback((next: BuyerDashboardPreferences) => {
    setPreferences(next);
    saveBuyerDashboardPreferences(next);
  }, []);

  return { preferences, applyPreferences };
}
