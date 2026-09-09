import type { LucideIcon } from 'lucide-react';
import type { ReactNode } from 'react';
import { Link } from 'react-router-dom';

import { adminKit } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';

/** Semantic bento icon tones (no rainbow); cycles brand / accent / neutral. */
export type BentoIconTone = 'brand' | 'accent' | 'neutral';

const TONE_STYLES: Record<BentoIconTone, string> = {
  brand: 'bg-primary/15 text-primary',
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
}: {
  title: string;
  children: ReactNode;
}) {
  return (
    <section >
      <h2 >{title}</h2>
      {children}
    </section>
  );
}

export function BentoGrid({ children }: { children: ReactNode }) {
  return (
    <div >{children}</div>
  );
}

export function BentoIconBadge({
  icon: Icon,
  tone = 'brand',
}: {
  icon: LucideIcon;
  tone?: BentoIconTone;
}) {
  return (
    <span
     
    >
      <Icon aria-hidden  strokeWidth={2} />
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
};

export function BentoCard({
  title,
  description,
  meta,
  icon: Icon,
  tone = 'brand',
  action,
  children,
}: BentoCardProps) {
  return (
    <article
     
    >
      <div >
        {Icon ? <BentoIconBadge icon={Icon} tone={tone} /> : <span  />}
        {action}
      </div>
      <div >
        <div >
          <div >{title}</div>
          {description ? (
            <p >
              {description}
            </p>
          ) : null}
          {children}
        </div>
      </div>
      {meta ? <footer >{meta}</footer> : null}
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
}: BentoCardProps & { path: string; actionLabel?: string }) {
  const resolvedTone = tone ?? bentoToneFromKey(path);

  return (
    <Link
     
      to={path}
    >
      <BentoCard
        action={
          <span
           
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
