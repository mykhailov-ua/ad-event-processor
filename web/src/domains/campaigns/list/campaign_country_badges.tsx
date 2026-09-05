import { MoreHorizontal } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { campaignCountriesOverflowPopoverClass } from '@/domains/campaigns/list/campaign_list_classes';
import { CountryFlagIcon } from '@/domains/campaigns/list/country_flag_icon';
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
    violet: 'border-violet-200 bg-violet-50 text-violet-800 dark:border-violet-900/50 dark:bg-violet-950/40 dark:text-violet-300',
    amber: 'border-amber-200 bg-amber-50 text-amber-800 dark:border-amber-900/50 dark:bg-amber-950/40 dark:text-amber-300',
    rose: 'border-rose-200 bg-rose-50 text-rose-800 dark:border-rose-900/50 dark:bg-rose-950/40 dark:text-rose-300',
    sky: 'border-sky-200 bg-sky-50 text-sky-800 dark:border-sky-900/50 dark:bg-sky-950/40 dark:text-sky-300',
    emerald: 'border-emerald-200 bg-emerald-50 text-emerald-800 dark:border-emerald-900/50 dark:bg-emerald-950/40 dark:text-emerald-300',
    indigo: 'border-indigo-200 bg-indigo-50 text-indigo-800 dark:border-indigo-900/50 dark:bg-indigo-950/40 dark:text-indigo-300',
    cyan: 'border-cyan-200 bg-cyan-50 text-cyan-800 dark:border-cyan-900/50 dark:bg-cyan-950/40 dark:text-cyan-300',
    orange: 'border-orange-200 bg-orange-50 text-orange-800 dark:border-orange-900/50 dark:bg-orange-950/40 dark:text-orange-300',
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
    return <CountryFlagIcon className="shrink-0" code={code} title={code} />;
  }

  return (
    <span
      className={cn(
        'inline-flex shrink-0 items-center gap-1 rounded border border-border px-1.5 py-0.5 text-xs tabular-nums',
        !listItem && 'max-w-full overflow-hidden',
        listItem && 'text-ui-mini',
        countryBadgeTone(code),
      )}
      title={code}
    >
      <CountryFlagIcon className="shrink-0" code={code} title={code} />
      <span className="whitespace-nowrap">{code}</span>
    </span>
  );
}

export type CampaignCountryBadgesProps = {
  countries?: readonly string[] | null;
  compact?: boolean;
  max?: number;
  overflowMenu?: boolean;
  className?: string;
};

export function CampaignCountryBadges({
  countries,
  compact = false,
  max = CAMPAIGN_COUNTRY_BADGES_MAX_VISIBLE,
  overflowMenu = false,
  className,
}: CampaignCountryBadgesProps) {
  const codes = Array.from(
    new Set(
      (countries ?? [])
        .map((code) => normalizeCountryCode(code))
        .filter((code): code is string => code != null),
    ),
  );
  if (codes.length === 0) {
    return null;
  }

  const visible = codes.slice(0, max);
  const overflow = codes.length - visible.length;
  const title = codes.join(', ');

  return (
    <span
      className={cn('inline-flex max-w-full flex-nowrap items-center gap-0.5 overflow-hidden', className)}
      title={overflow > 0 && !overflowMenu ? title : undefined}
    >
      {visible.map((code) => (
        <CountryCodeBadge key={code} code={code} compact={compact} />
      ))}
      {overflow > 0 ? (
        overflowMenu ? (
          <Popover>
            <PopoverTrigger asChild>
              <Button
                aria-label={`Show all ${codes.length} countries`}
                className="size-6 shrink-0 p-0 text-muted-foreground hover:text-foreground"
                type="button"
                variant="ghost"
                onClick={(event) => event.stopPropagation()}
              >
                <MoreHorizontal aria-hidden className="h-4 w-4" />
              </Button>
            </PopoverTrigger>
            <PopoverContent
              align="start"
              className={cn(campaignCountriesOverflowPopoverClass, 'p-0')}
              panelScroll="none"
              side="bottom"
              onClick={(event) => event.stopPropagation()}
            >
              <div className="flex flex-wrap gap-1 p-2">
                {codes.map((code) => (
                  <CountryCodeBadge key={code} code={code} compact={false} listItem />
                ))}
              </div>
            </PopoverContent>
          </Popover>
        ) : (
          <span className="inline-flex max-w-full items-center gap-0.5 overflow-hidden rounded border border-border bg-muted/50 px-1 text-ui-mini text-muted-foreground">
            +{overflow}
          </span>
        )
      ) : null}
    </span>
  );
}
