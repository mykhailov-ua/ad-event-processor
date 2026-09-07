import { Navigate } from 'react-router-dom';

import { useMeta } from '@/hooks/use_meta';
import { PageSkeleton } from '@/shell/page_skeleton';

// Legacy /setup URL: owner onboarding lives at /activate.
export function SetupPage() {
  const { bootstrapComplete, loading } = useMeta();

  if (loading) {
    return <PageSkeleton />;
  }

  if (bootstrapComplete) {
    return <Navigate replace to="/login" />;
  }

  return <Navigate replace to="/activate" />;
}
