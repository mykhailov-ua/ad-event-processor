import { MoreHorizontal } from 'lucide-react';

import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { campaignCountriesOverflowPopoverPanelClass } from '@/domains/campaigns/list/campaign_list_classes';
import { CountryFlagIcon } from '@/domains/campaigns/list/country_flag_icon';
import { directoryTableRowCopyButtonClass } from '@/shell/directory_table_row_actions';
import { cn } from '@/lib/utils';

const COUNTRY_TONE_BY_CODE: Record<string, string> = {
  US: 'violet',
  CA: 'violet',
  DE: 'amber',
  AT: 'amber',
  CH: 'amber',
  JP: 'rose',
  KR: 'rose',
  CN: 'rose',
  GB: 'sky',
  AU: 'sky',
  NZ: 'sky',
  IE: 'sky',
  BR: 'emerald',
  MX: 'emerald',
  AR: 'emerald',
  CO: 'emerald',
  FR: 'indigo',
  ES: 'indigo',
  IT: 'indigo',
  NL: 'indigo',
  PL: 'cyan',
  UA: 'cyan',
  TR: 'orange',
  IN: 'orange',
  ID: 'orange',
};

export const CAMPAIGN_COUNTRY_BADGES_MAX_VISIBLE = 3;

function normalizeCountryCode(raw: string): string | null {
  const code = raw.trim().toUpperCase();
  if (!/^[A-Z]{2}$/.test(code)) {
    return null;
  }
  return code;
}

function countryBadgeTone(code: string): string {
  const tone = COUNTRY_TONE_BY_CODE[code] ?? 'neutral';
  const tones: Record<string, string> = {
    violet: 'border-border bg-primary/10 text-primary',
    amber: 'border-admin-warn-border/60 bg-admin-warn-bg/50 text-admin-warn',
    rose: 'border-destructive/30 bg-destructive/10 text-destructive',
    sky: 'border-border bg-muted text-foreground',
    emerald: 'border-admin-status-active/30 bg-admin-status-active/15 text-admin-status-active',
    indigo: 'border-primary/25 bg-primary/10 text-primary',
    cyan: 'border-border bg-muted/80 text-muted-foreground',
    orange:
      'border-admin-status-scheduled/30 bg-admin-status-scheduled/15 text-admin-status-scheduled',
    neutral: 'border-border bg-muted/50 text-muted-foreground',
  };
  return tones[tone] ?? tones.neutral;
}

function CountryCodeBadge({
  code,
  compact,
  listItem = false,
}: {
  code: string;
  compact: boolean;
  listItem?: boolean;
}) {
  if (compact) {
    return <CountryFlagIcon  code={code} title={code} />;
  }

  return (
    <span
     
      title={code}
    >
      <CountryFlagIcon  code={code} title={code} />
      <span >{code}</span>
    </span>
  );
}

export type CampaignCountryBadgesProps = {
  countries?: readonly string[] | null;
  compact?: boolean;
  max?: number;
  overflowMenu?: boolean;
};

export function CampaignCountryBadges({
  countries,
  compact = false,
  max = CAMPAIGN_COUNTRY_BADGES_MAX_VISIBLE,
  overflowMenu = false,
}: CampaignCountryBadgesProps) {
  const codes = Array.from(
    new Set(
      (countries ?? [])
        .map((code) => normalizeCountryCode(code))
        .filter((code): code is string => code != null)
    )
  );
  if (codes.length === 0) {
    return null;
  }

  const visible = codes.slice(0, max);
  const overflow = codes.length - visible.length;
  const title = codes.join(', ');

  return (
    <span
     
      title={overflow > 0 && !overflowMenu ? title : undefined}
    >
      {visible.map((code) => (
        <CountryCodeBadge key={code} code={code} compact={compact} />
      ))}
      {overflow > 0 ? (
        overflowMenu ? (
          <Popover>
            <PopoverTrigger asChild>
              <button
                aria-label={`Show all ${codes.length} countries`}
               
                type="button"
                onClick={(event) => event.stopPropagation()}
              >
                <MoreHorizontal aria-hidden  />
              </button>
            </PopoverTrigger>
            <PopoverContent
              align="start"
             
              matchTriggerMinWidth={false}
              panelClassName={campaignCountriesOverflowPopoverPanelClass}
              panelScroll="none"
              side="bottom"
              onClick={(event) => event.stopPropagation()}
            >
              <div >
                {codes.map((code) => (
                  <CountryCodeBadge key={code} code={code} compact={false} listItem />
                ))}
              </div>
            </PopoverContent>
          </Popover>
        ) : (
          <span >
            +{overflow}
          </span>
        )
      ) : null}
    </span>
  );
}
