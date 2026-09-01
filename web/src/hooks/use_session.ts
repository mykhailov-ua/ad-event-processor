import { useContext } from 'react';

import { SessionContext, type SessionContextValue } from '@/providers/session_provider';

export function useSession(): SessionContextValue {
  const context = useContext(SessionContext);
  if (context === undefined) {
    throw new Error('useSession must be used within a SessionProvider');
  }
  return context;
}
