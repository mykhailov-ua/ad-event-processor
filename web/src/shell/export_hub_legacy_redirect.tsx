import { Navigate, useLocation } from 'react-router-dom';

export function PreserveSearchRedirect({ to }: { to: string }) {
  const location = useLocation();
  return <Navigate replace to={`${to}${location.search}`} />;
}

export function ReportJobsRoute() {
  const location = useLocation();
  const search = location.search;
  return <Navigate replace to={search ? `/exports${search}` : '/exports'} />;
}

export function BillingExportsRoute() {
  const location = useLocation();
  const search = location.search;
  return <Navigate replace to={search ? `/exports${search}` : '/exports'} />;
}
