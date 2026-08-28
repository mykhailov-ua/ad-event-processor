import { useSearchParams } from 'react-router-dom';
import { LoginForm } from '../ui/shell/login_form.js';

export function LoginPage() {
  const [searchParams] = useSearchParams();
  const reason = searchParams.get('reason');
  return <LoginForm reason={reason} gateLoading={false} />;
}
