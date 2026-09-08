import { useCallback, useState, useTransition } from 'react';

import {
  type BuyerDashboardPreferences,
  loadBuyerDashboardPreferences,
  saveBuyerDashboardPreferences,
} from '@/domains/dashboards/dashboard_preferences';

export function useBuyerDashboardPreferences() {
  const [preferences, setPreferences] = useState<BuyerDashboardPreferences>(() =>
    loadBuyerDashboardPreferences()
  );
  const [, startPreferenceTransition] = useTransition();

  const applyPreferences = useCallback(
    (next: BuyerDashboardPreferences) => {
      startPreferenceTransition(() => {
        setPreferences(next);
        saveBuyerDashboardPreferences(next);
      });
    },
    [startPreferenceTransition]
  );

  return { preferences, applyPreferences };
}
