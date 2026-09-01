import { useContext } from 'react';

import { MetaContext, type MetaContextValue } from '@/providers/meta_provider';

export function useMeta(): MetaContextValue {
  const value = useContext(MetaContext);
  if (!value) {
    throw new Error('useMeta must be used within MetaProvider');
  }
  return value;
}
