import { Link } from 'react-router-dom';

import { SUPPLY_PREVIEW_ADS_TXT_PATH, SUPPLY_PREVIEW_SELLERS_JSON_PATH } from '@/api/supply_api';
import { PageChrome } from '@/shell/page_chrome';
import { EmptyState } from '@/shell/empty_state';
import { PageSkeleton } from '@/shell/page_skeleton';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import {
  DirectoryTable,
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from '@/shell/directory_table';
import type { AdsTxtEntry, Seller, SupplyExportPath, SupplyValidation } from '@/api/types';
import { CreativeDirectoryStack } from '@/domains/creative/creative_directory_stack';
import { creativePanelError } from '@/domains/creative/creative_nav';
import { ActionLinksBand, MetaLinksBand, TableHost } from '@/shell/ui_bands';

export type SupplyHubProps = {
  sellers: Seller[];
  adsTxt: AdsTxtEntry[];
  exportPath: SupplyExportPath | undefined;
  validation: SupplyValidation | undefined;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  previewSellersJson?: string;
  previewAdsTxt?: string;
  previewSellersError?: Error;
  previewAdsTxtError?: Error;
  loadingSellersPreview?: boolean;
  loadingAdsTxtPreview?: boolean;
  onLoadSellersPreview?: () => void;
  onLoadAdsTxtPreview?: () => void;
};

export function SupplyHub({
  sellers,
  adsTxt,
  exportPath,
  validation,
  fetching,
  error,
  hasSnapshot,
  previewSellersJson,
  previewAdsTxt,
  previewSellersError,
  previewAdsTxtError,
  loadingSellersPreview = false,
  loadingAdsTxtPreview = false,
  onLoadSellersPreview,
  onLoadAdsTxtPreview,
}: SupplyHubProps) {
  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return (
      <PageChrome title="Supply">
        <CreativeDirectoryStack>
          {creativePanelError(error, 'Could not load supply data')}
        </CreativeDirectoryStack>
      </PageChrome>
    );
  }

  return (
    <PageChrome title="Supply">
      <CreativeDirectoryStack>
      <section className="grid gap-2">
        <h2 className="text-base font-semibold">Server previews</h2>
        <ul className="list-inside list-disc text-sm">
          <li>
            <a
              className="underline"
              href={SUPPLY_PREVIEW_SELLERS_JSON_PATH}
              target="_blank"
              rel="noreferrer"
            >
              sellers.json preview
            </a>
          </li>
          <li>
            <a
              className="underline"
              href={SUPPLY_PREVIEW_ADS_TXT_PATH}
              target="_blank"
              rel="noreferrer"
            >
              ads.txt preview
            </a>
          </li>
        </ul>
        <ActionLinksBand>
          {onLoadSellersPreview ? (
            <Button
              disabled={loadingSellersPreview}
              onClick={onLoadSellersPreview}
              type="button"
              variant="outline"
            >
              {loadingSellersPreview ? 'Loading sellers.json...' : 'Load sellers.json inline'}
            </Button>
          ) : null}
          {onLoadAdsTxtPreview ? (
            <Button
              disabled={loadingAdsTxtPreview}
              onClick={onLoadAdsTxtPreview}
              type="button"
              variant="outline"
            >
              {loadingAdsTxtPreview ? 'Loading ads.txt...' : 'Load ads.txt inline'}
            </Button>
          ) : null}
        </ActionLinksBand>
        {previewSellersError
          ? creativePanelError(previewSellersError, 'Could not load sellers.json preview')
          : null}
        {previewAdsTxtError
          ? creativePanelError(previewAdsTxtError, 'Could not load ads.txt preview')
          : null}
        {previewSellersJson ? (
          <pre className="max-h-64 overflow-auto border bg-muted/40 p-3 font-mono text-xs whitespace-pre-wrap">
            {previewSellersJson}
          </pre>
        ) : null}
        {previewAdsTxt ? (
          <pre className="max-h-64 overflow-auto border bg-muted/40 p-3 font-mono text-xs whitespace-pre-wrap">
            {previewAdsTxt}
          </pre>
        ) : null}
        {exportPath?.path ? (
          <p className="text-sm text-muted-foreground">
            Nginx export path:{' '}
            <span className="font-mono text-xs text-foreground">{exportPath.path}</span>
          </p>
        ) : null}
      </section>

      {validation ? (
        <section className="grid gap-2">
          <h2 className="text-base font-semibold">Validation</h2>
          <ActionLinksBand className="text-sm">
            <Badge variant={validation.sellers_json_valid ? 'default' : 'destructive'}>
              sellers.json {validation.sellers_json_valid ? 'valid' : 'invalid'}
            </Badge>
            <Badge variant={validation.ads_txt_valid ? 'default' : 'destructive'}>
              ads.txt {validation.ads_txt_valid ? 'valid' : 'invalid'}
            </Badge>
            <Badge variant="outline">{validation.sellers_count} sellers</Badge>
            <Badge variant="outline">{validation.ads_txt_line_count} ads.txt lines</Badge>
          </ActionLinksBand>
          {(validation.issues ?? []).length > 0 ? (
            <ul className="list-inside list-disc text-sm text-muted-foreground">
              {(validation.issues ?? []).map((issue) => (
                <li key={issue}>{issue}</li>
              ))}
            </ul>
          ) : null}
        </section>
      ) : null}

      <section className="grid gap-2">
        <h2 className="text-base font-semibold">Sellers</h2>
        {sellers.length === 0 ? (
          <EmptyState title="No sellers" description="Supply sellers table is empty." />
        ) : (
          <TableHost>
            <DirectoryTable nested>
            <TableHeader>
              <TableRow>
                <DirectoryTableHead>Seller ID</DirectoryTableHead>
                <DirectoryTableHead>Domain</DirectoryTableHead>
                <DirectoryTableHead>Type</DirectoryTableHead>
                <DirectoryTableHead>Name</DirectoryTableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {sellers.map((row) => (
                <TableRow key={row.id}>
                  <TableCell className="font-mono text-xs">{row.seller_id}</TableCell>
                  <TableCell>{row.domain}</TableCell>
                  <TableCell>{row.seller_type}</TableCell>
                  <TableCell>{row.name}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </DirectoryTable>
          </TableHost>
        )}
      </section>

      <section className="grid gap-2">
        <h2 className="text-base font-semibold">ads.txt rows</h2>
        {adsTxt.length === 0 ? (
          <EmptyState title="No ads.txt rows" description="Supply ads.txt table is empty." />
        ) : (
          <TableHost>
            <DirectoryTable nested>
            <TableHeader>
              <TableRow>
                <DirectoryTableHead>Domain</DirectoryTableHead>
                <DirectoryTableHead>Account</DirectoryTableHead>
                <DirectoryTableHead>Relationship</DirectoryTableHead>
                <DirectoryTableHead>Order</DirectoryTableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {adsTxt.map((row) => (
                <TableRow key={row.id}>
                  <TableCell>{row.domain}</TableCell>
                  <TableCell className="font-mono text-xs">{row.publisher_account_id}</TableCell>
                  <TableCell>{row.relationship}</TableCell>
                  <TableCell>{row.sort_order}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </DirectoryTable>
          </TableHost>
        )}
      </section>

      <MetaLinksBand>
        <Link to="/supply/sellers">Sellers sub-route</Link>
        <span aria-hidden>*</span>
        <Link to="/supply/ads-txt">ads.txt sub-route</Link>
      </MetaLinksBand>

      {error && hasSnapshot ? creativePanelError(error, 'Refresh failed') : null}
      </CreativeDirectoryStack>
    </PageChrome>
  );
}
