import { newRandomUuid } from '@/lib/uuid';

export type LanderBlockType = 'hero' | 'cta' | 'image' | 'html';

export type LanderHeroBlock = {
  id: string;
  type: 'hero';
  title: string;
  subtitle: string;
};

export type LanderCtaBlock = {
  id: string;
  type: 'cta';
  label: string;
  href: string;
};

export type LanderImageBlock = {
  id: string;
  type: 'image';
  src: string;
  alt: string;
};

export type LanderHtmlBlock = {
  id: string;
  type: 'html';
  content: string;
};

export type LanderBlock = LanderHeroBlock | LanderCtaBlock | LanderImageBlock | LanderHtmlBlock;

export function newLanderBlock(type: LanderBlockType): LanderBlock {
  const id = newRandomUuid();
  switch (type) {
    case 'hero':
      return { id, type: 'hero', title: 'Headline', subtitle: 'Supporting copy' };
    case 'cta':
      return { id, type: 'cta', label: 'Continue', href: 'https://example.com/click' };
    case 'image':
      return { id, type: 'image', src: 'https://example.com/hero.jpg', alt: 'Hero image' };
    case 'html':
      return { id, type: 'html', content: '<p>Custom HTML block</p>' };
    default:
      return { id, type: 'hero', title: 'Headline', subtitle: '' };
  }
}

export const LANDER_BLOCK_OPTIONS: { value: LanderBlockType; label: string }[] = [
  { value: 'hero', label: 'Hero' },
  { value: 'cta', label: 'CTA button' },
  { value: 'image', label: 'Image' },
  { value: 'html', label: 'Raw HTML' },
];
