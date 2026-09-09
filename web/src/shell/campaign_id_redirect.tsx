import { Navigate, useLocation, useParams } from 'react-router-dom';

export function CampaignIdRedirect() {
  const { id } = useParams();
  const location = useLocation();

  if (!id) {
    return <Navigate replace to={`/campaigns${location.search}`} />;
  }

  return (
    <Navigate replace to={`/campaigns/${encodeURIComponent(id)}/edit${location.search}`} />
  );
}
