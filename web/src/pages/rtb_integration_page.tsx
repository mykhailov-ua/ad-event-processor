import { useCallback } from 'react';
import { useSearchParams } from 'react-router-dom';
import { parseRtbDetailTab, type RtbDetailTab } from '../helpers/rtb_api.js';
import { RtbIntegrationDetail } from '../ui/rtb/rtb_integration_detail.js';

export function RtbIntegrationPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const tab = parseRtbDetailTab(searchParams.get('tab'));

  const onTabChange = useCallback(
    (next: RtbDetailTab) => {
      const params = new URLSearchParams(searchParams);
      if (next === 'profile') {
        params.delete('tab');
      } else {
        params.set('tab', next);
      }
      setSearchParams(params, { replace: true });
    },
    [searchParams, setSearchParams]
  );

  return <RtbIntegrationDetail tab={tab} onTabChange={onTabChange} />;
}
