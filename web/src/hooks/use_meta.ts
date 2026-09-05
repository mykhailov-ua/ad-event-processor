import { useContext } from 'react';

import { MetaContext, type MetaContextValue } from '@/context/meta_context';

export function useMeta(): MetaContextValue {
  const value = useContext(MetaContext);
  if (!value) {
    throw new Error('useMeta must be used within MetaProvider');
  }
  return value;
}
