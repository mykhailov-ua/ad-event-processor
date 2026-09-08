import type { LucideIcon } from 'lucide-react';
import type { ReactNode } from 'react';
import { Link } from 'react-router-dom';

import { adminKit } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';

/** Semantic bento icon tones (no rainbow); cycles brand / accent / neutral. */
export type BentoIconTone = 'brand' | 'accent' | 'neutral';

const TONE_STYLES: Record<BentoIconTone, string> = {
  brand: 'bg-admin-brand/15 text-admin-brand',
  accent: 'bg-chart-1/15 text-chart-1',
  neutral: 'bg-muted text-muted-foreground',
};

const TONE_CYCLE: BentoIconTone[] = ['brand', 'accent', 'neutral'];

export function bentoToneFromKey(key: string): BentoIconTone {
  let hash = 0;
  for (let i = 0; i < key.length; i++) {
    hash = (hash * 31 + key.charCodeAt(i)) >>> 0;
  }
  return TONE_CYCLE[hash % TONE_CYCLE.length];
}

export function BentoSection({
  title,
  children,
  className,
}: {
  title: string;
  children: ReactNode;
  className?: string;
}) {
  return (
    <section className={cn('grid gap-4', className)}>
      <h2 className="text-sm font-medium text-muted-foreground">{title}</h2>
      {children}
    </section>
  );
}

export function BentoGrid({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <div className={cn('grid gap-4 sm:grid-cols-2 xl:grid-cols-3', className)}>{children}</div>
  );
}

export function BentoIconBadge({
  icon: Icon,
  tone = 'brand',
  className,
}: {
  icon: LucideIcon;
  tone?: BentoIconTone;
  className?: string;
}) {
  return (
    <span
      className={cn(
        'inline-flex size-8 shrink-0 items-center justify-center',
        adminKit.panelRadius,
        TONE_STYLES[tone],
        className
      )}
    >
      <Icon aria-hidden className="size-4" strokeWidth={2} />
    </span>
  );
}

export type BentoCardProps = {
  title: ReactNode;
  description?: ReactNode;
  meta?: ReactNode;
  icon?: LucideIcon;
  tone?: BentoIconTone;
  action?: ReactNode;
  children?: ReactNode;
  className?: string;
};

export function BentoCard({
  title,
  description,
  meta,
  icon: Icon,
  tone = 'brand',
  action,
  children,
  className,
}: BentoCardProps) {
  return (
    <article
      className={cn(
        'ui-bento-card group ui-surface-raised grid h-full grid-rows-[auto_minmax(0,1fr)_auto] gap-4 p-5 transition-colors duration-200 hover:border-border/80',
        className
      )}
    >
      <div className="grid grid-cols-[1fr_auto] items-start gap-3">
        {Icon ? <BentoIconBadge icon={Icon} tone={tone} /> : <span className="size-8 shrink-0" />}
        {action}
      </div>
      <div className="grid min-w-0 flex-1 gap-2">
        <div className="text-base font-medium leading-snug tracking-tight">{title}</div>
        {description ? (
          <p className="whitespace-normal text-sm leading-relaxed text-muted-foreground">
            {description}
          </p>
        ) : null}
        {children}
      </div>
      {meta ? <footer className="text-xs text-muted-foreground">{meta}</footer> : null}
    </article>
  );
}

export function BentoLinkCard({
  path,
  title,
  description,
  meta,
  icon,
  tone,
  actionLabel = 'Open',
}: {
  path: string;
  title: string;
  description: string;
  meta?: string;
  icon?: LucideIcon;
  tone?: BentoIconTone;
  actionLabel?: string;
}) {
  const resolvedTone = tone ?? bentoToneFromKey(path);

  return (
    <Link
      className={cn(
        'block h-full outline-none focus-visible:ring-2 focus-visible:ring-ring',
        adminKit.panelRadius
      )}
      to={path}
    >
      <BentoCard
        action={
          <span
            className={cn(
              'bg-primary/10 px-2.5 py-1 text-xs text-primary transition-colors group-hover:bg-primary/20',
              adminKit.pillRadius
            )}
          >
            {actionLabel}
          </span>
        }
        description={description}
        icon={icon}
        meta={meta}
        title={title}
        tone={resolvedTone}
      />
    </Link>
  );
}
