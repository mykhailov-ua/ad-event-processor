import type { LucideIcon } from 'lucide-react';
import type { ReactNode } from 'react';
import { Link } from 'react-router-dom';

import { cn } from '@/lib/utils';

export type BentoIconTone = 'blue' | 'violet' | 'pink' | 'amber' | 'cyan' | 'emerald';

const TONE_STYLES: Record<BentoIconTone, string> = {
  blue: 'bg-sky-500/15 text-sky-400',
  violet: 'bg-violet-500/15 text-violet-400',
  pink: 'bg-pink-500/15 text-pink-400',
  amber: 'bg-amber-500/15 text-amber-400',
  cyan: 'bg-cyan-500/15 text-cyan-400',
  emerald: 'bg-emerald-500/15 text-emerald-400',
};

const TONE_CYCLE: BentoIconTone[] = ['blue', 'violet', 'pink', 'amber', 'cyan', 'emerald'];

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

export function BentoGrid({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  return (
    <div className={cn('grid gap-4 sm:grid-cols-2 xl:grid-cols-3', className)}>{children}</div>
  );
}

export function BentoIconBadge({
  icon: Icon,
  tone = 'blue',
  className,
}: {
  icon: LucideIcon;
  tone?: BentoIconTone;
  className?: string;
}) {
  return (
    <span
      className={cn(
        'inline-flex size-8 shrink-0 items-center justify-center rounded-xl',
        TONE_STYLES[tone],
        className,
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
  tone = 'blue',
  action,
  children,
  className,
}: BentoCardProps) {
  return (
    <article className={cn('ui-bento-card group flex h-full flex-col gap-4 rounded-2xl border border-border/40 bg-card/50 p-5 transition-colors duration-200 hover:border-border/80', className)}>
      <div className="flex items-start justify-between gap-3">
        {Icon ? <BentoIconBadge icon={Icon} tone={tone} /> : <span className="size-8 shrink-0" />}
        {action}
      </div>
      <div className="grid min-w-0 flex-1 gap-2">
        <div className="text-base font-medium leading-snug tracking-tight">{title}</div>
        {description ? (
          <p className="line-clamp-2 text-sm leading-relaxed text-muted-foreground">{description}</p>
        ) : null}
        {children}
      </div>
      {meta ? <footer className="mt-auto text-xs text-muted-foreground">{meta}</footer> : null}
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
    <Link className="block h-full rounded-2xl outline-none focus-visible:ring-2 focus-visible:ring-ring" to={path}>
      <BentoCard
        action={
          <span className="rounded-full bg-secondary px-2.5 py-1 text-xs text-muted-foreground transition-colors group-hover:text-foreground">
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
