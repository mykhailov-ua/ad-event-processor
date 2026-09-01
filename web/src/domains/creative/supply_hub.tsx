import { Link } from 'react-router-dom';

import {
  SUPPLY_PREVIEW_ADS_TXT_PATH,
  SUPPLY_PREVIEW_SELLERS_JSON_PATH,
} from '@/api/supply_api';
import { PageChrome } from '@/components/system/page_chrome';
import { EmptyState } from '@/components/system/empty_state';
import { PageSkeleton } from '@/components/system/page_skeleton';
import { Badge } from '@/components/ui/badge';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import type {
  AdsTxtEntry,
  Seller,
  SupplyExportPath,
  SupplyValidation,
} from '@/api/types';
import { CreativeNav, creativePanelError } from '@/domains/creative/creative_nav';

export type SupplyHubProps = {
  sellers: Seller[];
  adsTxt: AdsTxtEntry[];
  exportPath: SupplyExportPath | undefined;
  validation: SupplyValidation | undefined;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
};

export function SupplyHub({
  sellers,
  adsTxt,
  exportPath,
  validation,
  fetching,
  error,
  hasSnapshot,
}: SupplyHubProps) {
  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return (
      <PageChrome title="Supply">
        <CreativeNav />
        {creativePanelError(error, 'Could not load supply data')}
      </PageChrome>
    );
  }

  return (
    <PageChrome title="Supply">
      <CreativeNav />

      <section className="grid gap-2">
        <h2 className="text-base font-semibold">Server previews</h2>
        <ul className="list-inside list-disc text-sm">
          <li>
            <a className="underline" href={SUPPLY_PREVIEW_SELLERS_JSON_PATH} target="_blank" rel="noreferrer">
              sellers.json preview
            </a>
          </li>
          <li>
            <a className="underline" href={SUPPLY_PREVIEW_ADS_TXT_PATH} target="_blank" rel="noreferrer">
              ads.txt preview
            </a>
          </li>
        </ul>
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
          <div className="flex flex-wrap gap-2 text-sm">
            <Badge variant={validation.sellers_json_valid ? 'default' : 'destructive'}>
              sellers.json {validation.sellers_json_valid ? 'valid' : 'invalid'}
            </Badge>
            <Badge variant={validation.ads_txt_valid ? 'default' : 'destructive'}>
              ads.txt {validation.ads_txt_valid ? 'valid' : 'invalid'}
            </Badge>
            <Badge variant="outline">{validation.sellers_count} sellers</Badge>
            <Badge variant="outline">{validation.ads_txt_line_count} ads.txt lines</Badge>
          </div>
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
          <div className="ui-table-frame">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Seller ID</TableHead>
                  <TableHead>Domain</TableHead>
                  <TableHead>Type</TableHead>
                  <TableHead>Name</TableHead>
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
            </Table>
          </div>
        )}
      </section>

      <section className="grid gap-2">
        <h2 className="text-base font-semibold">ads.txt rows</h2>
        {adsTxt.length === 0 ? (
          <EmptyState title="No ads.txt rows" description="Supply ads.txt table is empty." />
        ) : (
          <div className="ui-table-frame">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Domain</TableHead>
                  <TableHead>Account</TableHead>
                  <TableHead>Relationship</TableHead>
                  <TableHead>Order</TableHead>
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
            </Table>
          </div>
        )}
      </section>

      <p className="text-sm">
        <Link className="text-muted-foreground hover:underline" to="/supply/sellers">
          Sellers sub-route
        </Link>
        {' * '}
        <Link className="text-muted-foreground hover:underline" to="/supply/ads-txt">
          ads.txt sub-route
        </Link>
      </p>

      {error && hasSnapshot ? creativePanelError(error, 'Refresh failed') : null}
    </PageChrome>
  );
}
