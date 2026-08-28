import { useState } from 'react';
import { Link } from 'react-router-dom';
import * as auth from '../../helpers/auth.js';
import type {
  SupplyAdsTxtEntry,
  SupplyExportPath,
  SupplySeller,
  SupplyValidation,
} from '../../helpers/integrations_api.js';
import { can } from '../../helpers/permissions.js';
import { Button } from '../system/button.js';
import { EmptyState } from '../system/empty_state.js';
import { ErrorBlock } from '../system/error_block.js';
import { PageChrome } from '../system/page_chrome.js';
import shared from '../integrations/integrations_shared.module.css';
import styles from './supply.module.css';

export type SupplyPanelProps = {
  sellers: SupplySeller[];
  adsTxt: SupplyAdsTxtEntry[];
  validation: SupplyValidation | null;
  exportPath: SupplyExportPath | null;
  sellersPreview: string;
  adsTxtPreview: string;
  previewTab: 'sellers' | 'ads_txt';
  loading: boolean;
  error: unknown;
  busy: boolean;
  onPreviewTabChange: (tab: 'sellers' | 'ads_txt') => void;
  onReloadValidation: () => void;
  onCreateSeller: (body: {
    seller_id: string;
    domain: string;
    seller_type: string;
    name: string;
    is_confidential: boolean;
  }) => void;
  onDeleteSeller: (id: number) => void;
  onCreateAdsTxt: (body: {
    domain: string;
    publisher_account_id: string;
    relationship: string;
    cert_authority_id?: string;
    sort_order?: number;
  }) => void;
  onDeleteAdsTxt: (id: number) => void;
};

export function SupplyPanel({
  sellers,
  adsTxt,
  validation,
  exportPath,
  sellersPreview,
  adsTxtPreview,
  previewTab,
  loading,
  error,
  busy,
  onPreviewTabChange,
  onReloadValidation,
  onCreateSeller,
  onDeleteSeller,
  onCreateAdsTxt,
  onDeleteAdsTxt,
}: SupplyPanelProps) {
  const permissions = auth.getUser()?.permissions ?? [];
  const canRead = can(permissions, 'settings:read');
  const canWrite = can(permissions, 'settings:write');

  const [sellerId, setSellerId] = useState('');
  const [sellerDomain, setSellerDomain] = useState('');
  const [sellerType, setSellerType] = useState('PUBLISHER');
  const [sellerName, setSellerName] = useState('');
  const [adsDomain, setAdsDomain] = useState('');
  const [adsAccount, setAdsAccount] = useState('');
  const [adsRelationship, setAdsRelationship] = useState('DIRECT');

  if (!canRead) {
    return <ErrorBlock error={new Error('forbidden')} fallbackTitle="Supply access denied" />;
  }

  if (error) {
    return <ErrorBlock error={error} fallbackTitle="Failed to load supply data" />;
  }

  return (
    <div className={shared.panelRoot} data-testid="integrations-supply-page">
      <PageChrome
        title="Supply files"
        badge={
          <Link to="/integrations" className={shared.bannerLink}>
            All integrations
          </Link>
        }
      />

      <section>
        <h2 className={shared.sectionTitle}>Validation</h2>
        {validation ? (
          <div className={shared.formStack}>
            <p className={shared.hint}>
              sellers.json valid: {validation.sellers_json_valid ? 'yes' : 'no'} (
              {validation.sellers_count ?? 0} sellers)
            </p>
            <p className={shared.hint}>
              ads.txt valid: {validation.ads_txt_valid ? 'yes' : 'no'} (
              {validation.ads_txt_line_count ?? 0} lines)
            </p>
            {validation.issues && validation.issues.length > 0 ? (
              <ul className={styles.validationList}>
                {validation.issues.map((issue) => (
                  <li key={issue}>{issue}</li>
                ))}
              </ul>
            ) : null}
          </div>
        ) : (
          <p className={shared.hint}>No validation snapshot loaded.</p>
        )}
        <div className={shared.toolbar}>
          <Button size="sm" variant="secondary" onClick={onReloadValidation}>
            Refresh validation
          </Button>
          {exportPath?.path ? (
            <span className={shared.hint}>Export path: {exportPath.path}</span>
          ) : null}
        </div>
      </section>

      <section>
        <h2 className={shared.sectionTitle}>Preview</h2>
        <div className={styles.tabs}>
          <Button
            size="sm"
            variant={previewTab === 'sellers' ? 'primary' : 'secondary'}
            onClick={() => onPreviewTabChange('sellers')}
          >
            sellers.json
          </Button>
          <Button
            size="sm"
            variant={previewTab === 'ads_txt' ? 'primary' : 'secondary'}
            onClick={() => onPreviewTabChange('ads_txt')}
          >
            ads.txt
          </Button>
        </div>
        <pre className={shared.codePreview}>
          {previewTab === 'sellers' ? sellersPreview : adsTxtPreview}
        </pre>
      </section>

      <section>
        <h2 className={shared.sectionTitle}>Sellers</h2>
        <div className={shared.gridTable} role="grid">
          <div className={`${shared.gridHeader} ${styles.colsSellers}`} role="row">
            <span className={shared.gridCell} role="columnheader">
              ID
            </span>
            <span className={shared.gridCell} role="columnheader">
              Seller ID
            </span>
            <span className={shared.gridCell} role="columnheader">
              Domain
            </span>
            <span className={shared.gridCell} role="columnheader">
              Type
            </span>
            <span className={shared.gridCell} role="columnheader">
              Name
            </span>
            <span className={shared.gridCell} role="columnheader">
              Confidential
            </span>
            <span className={shared.gridCell} role="columnheader">
              Action
            </span>
          </div>
          {sellers.length === 0 && !loading ? (
            <EmptyState message="No sellers configured." />
          ) : (
            sellers.map((row) => (
              <div
                key={row.id}
                className={`${shared.gridRow} ${styles.colsSellers}`}
                role="row"
              >
                <span className={shared.gridCell} role="gridcell">
                  {row.id}
                </span>
                <span className={shared.gridCell} role="gridcell">
                  {row.seller_id}
                </span>
                <span className={shared.gridCell} role="gridcell">
                  {row.domain}
                </span>
                <span className={shared.gridCell} role="gridcell">
                  {row.seller_type}
                </span>
                <span className={shared.gridCell} role="gridcell">
                  {row.name}
                </span>
                <span className={shared.gridCell} role="gridcell">
                  {row.is_confidential ? 'yes' : 'no'}
                </span>
                <span className={shared.gridCell} role="gridcell">
                  {canWrite && row.id != null ? (
                    <Button
                      size="sm"
                      variant="secondary"
                      disabled={busy}
                      onClick={() => onDeleteSeller(row.id as number)}
                    >
                      Delete
                    </Button>
                  ) : (
                    '-'
                  )}
                </span>
              </div>
            ))
          )}
        </div>
        {canWrite ? (
          <div className={shared.formStack}>
            <label className={shared.field}>
              <span className={shared.fieldLabel}>Seller ID</span>
              <input
                className={shared.textInput}
                value={sellerId}
                onChange={(event) => setSellerId(event.target.value)}
              />
            </label>
            <label className={shared.field}>
              <span className={shared.fieldLabel}>Domain</span>
              <input
                className={shared.textInput}
                value={sellerDomain}
                onChange={(event) => setSellerDomain(event.target.value)}
              />
            </label>
            <label className={shared.field}>
              <span className={shared.fieldLabel}>Type</span>
              <input
                className={shared.textInput}
                value={sellerType}
                onChange={(event) => setSellerType(event.target.value)}
              />
            </label>
            <label className={shared.field}>
              <span className={shared.fieldLabel}>Name</span>
              <input
                className={shared.textInput}
                value={sellerName}
                onChange={(event) => setSellerName(event.target.value)}
              />
            </label>
            <Button
              size="sm"
              variant="primary"
              disabled={busy}
              onClick={() =>
                onCreateSeller({
                  seller_id: sellerId.trim(),
                  domain: sellerDomain.trim(),
                  seller_type: sellerType.trim(),
                  name: sellerName.trim(),
                  is_confidential: false,
                })
              }
            >
              Add seller
            </Button>
          </div>
        ) : null}
      </section>

      <section>
        <h2 className={shared.sectionTitle}>ads.txt entries</h2>
        <div className={shared.gridTable} role="grid">
          <div className={`${shared.gridHeader} ${styles.colsAdsTxt}`} role="row">
            <span className={shared.gridCell} role="columnheader">
              ID
            </span>
            <span className={shared.gridCell} role="columnheader">
              Domain
            </span>
            <span className={shared.gridCell} role="columnheader">
              Account
            </span>
            <span className={shared.gridCell} role="columnheader">
              Relationship
            </span>
            <span className={shared.gridCell} role="columnheader">
              Order
            </span>
            <span className={shared.gridCell} role="columnheader">
              Action
            </span>
          </div>
          {adsTxt.length === 0 && !loading ? (
            <EmptyState message="No ads.txt lines configured." />
          ) : (
            adsTxt.map((row) => (
              <div key={row.id} className={`${shared.gridRow} ${styles.colsAdsTxt}`} role="row">
                <span className={shared.gridCell} role="gridcell">
                  {row.id}
                </span>
                <span className={shared.gridCell} role="gridcell">
                  {row.domain}
                </span>
                <span className={shared.gridCell} role="gridcell">
                  {row.publisher_account_id}
                </span>
                <span className={shared.gridCell} role="gridcell">
                  {row.relationship}
                </span>
                <span className={shared.gridCell} role="gridcell">
                  {row.sort_order}
                </span>
                <span className={shared.gridCell} role="gridcell">
                  {canWrite && row.id != null ? (
                    <Button
                      size="sm"
                      variant="secondary"
                      disabled={busy}
                      onClick={() => onDeleteAdsTxt(row.id as number)}
                    >
                      Delete
                    </Button>
                  ) : (
                    '-'
                  )}
                </span>
              </div>
            ))
          )}
        </div>
        {canWrite ? (
          <div className={shared.formStack}>
            <label className={shared.field}>
              <span className={shared.fieldLabel}>Domain</span>
              <input
                className={shared.textInput}
                value={adsDomain}
                onChange={(event) => setAdsDomain(event.target.value)}
              />
            </label>
            <label className={shared.field}>
              <span className={shared.fieldLabel}>Publisher account ID</span>
              <input
                className={shared.textInput}
                value={adsAccount}
                onChange={(event) => setAdsAccount(event.target.value)}
              />
            </label>
            <label className={shared.field}>
              <span className={shared.fieldLabel}>Relationship</span>
              <input
                className={shared.textInput}
                value={adsRelationship}
                onChange={(event) => setAdsRelationship(event.target.value)}
              />
            </label>
            <Button
              size="sm"
              variant="primary"
              disabled={busy}
              onClick={() =>
                onCreateAdsTxt({
                  domain: adsDomain.trim(),
                  publisher_account_id: adsAccount.trim(),
                  relationship: adsRelationship.trim(),
                })
              }
            >
              Add ads.txt line
            </Button>
          </div>
        ) : null}
      </section>
    </div>
  );
}
