import { adminKit } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';

/** Public path copied to dist by build_lib.copyProductAvatarAssets. */
export const PRODUCT_AVATAR_SRC = '/src/assets/product_avatar.svg';

export type ProductAvatarSize = 'sm' | 'md' | 'lg';

const SIZE_CLASS: Record<ProductAvatarSize, string> = {
  sm: 'h-6 w-6',
  md: 'h-8 w-8',
  lg: 'h-12 w-12',
};

export type ProductAvatarProps = {
  size?: ProductAvatarSize;
  className?: string;
  framed?: boolean;
};

/** Scalable product avatar (SVG). Sizes: sm 24px, md 32px, lg 48px. */
export function ProductAvatar({ size = 'md', className, framed = false }: ProductAvatarProps) {
  return (
    <span
      className={cn(
        'inline-flex shrink-0 items-center justify-center overflow-hidden',
        SIZE_CLASS[size],
        framed && cn('bg-primary text-primary-foreground', adminKit.controlRadius),
        className
      )}
    >
      <img
        alt=""
        aria-hidden
        className={cn('h-full w-full object-cover', framed && adminKit.controlRadius)}
        decoding="async"
        draggable={false}
        src={PRODUCT_AVATAR_SRC}
      />
    </span>
  );
}
