import * as auth from '../helpers/auth.js';
import { can } from '../helpers/permissions.js';
import { CustomerApiKeysSection } from '../components/customer_api_keys_section.js';

export function SelfServeApiKeysPage() {
  const user = auth.getUser();
  const canCreate = can(user?.permissions ?? [], 'campaigns:write');

  return (
    <section className="stack" data-testid="selfserve-api-keys-page">
      <div className="page-header">
        <h1 className="page-header__title">API keys</h1>
        <p className="page-header__desc">
          Programmatic access for your account. Keys are shown once at creation.
        </p>
      </div>
      <CustomerApiKeysSection canCreate={canCreate} />
    </section>
  );
}
