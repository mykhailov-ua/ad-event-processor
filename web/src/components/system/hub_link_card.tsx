import type { LucideIcon } from 'lucide-react';
import type { ReactNode } from 'react';

import {
  BentoGrid,
  BentoLinkCard,
  type BentoIconTone,
  bentoToneFromKey,
} from '@/components/system/bento_card';
import { cn } from '@/lib/utils';

export type HubLinkItem = {
  path: string;
  title: string;
  description: string;
  icon?: LucideIcon;
  tone?: BentoIconTone;
  meta?: string;
};

export function HubLinkGrid({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  return <BentoGrid className={className}>{children}</BentoGrid>;
}

export function HubLinkCard({ path, title, description, icon, tone, meta }: HubLinkItem) {
  return (
    <BentoLinkCard
      description={description}
      icon={icon}
      meta={meta}
      path={path}
      title={title}
      tone={tone ?? bentoToneFromKey(path)}
    />
  );
}
