import * as React from 'react';

import { cn } from '@/lib/utils';

export function mergeRefs<T>(...refs: Array<React.Ref<T> | undefined>) {
  return (value: T) => {
    for (const ref of refs) {
      if (typeof ref === 'function') {
        ref(value);
      } else if (ref != null) {
        (ref as React.MutableRefObject<T>).current = value;
      }
    }
  };
}

export function composeHandlers<E>(
  ...handlers: Array<((event: E) => void) | undefined>
) {
  return (event: E) => {
    for (const handler of handlers) {
      handler?.(event);
    }
  };
}

export type SlotProps = React.HTMLAttributes<HTMLElement> & {
  children?: React.ReactNode;
};

export const Slot = React.forwardRef<HTMLElement, SlotProps>(function Slot(
  { children, className, ...props },
  forwardedRef,
) {
  if (!React.isValidElement(children)) {
    return children ?? null;
  }

  const child = children as React.ReactElement<Record<string, unknown>>;
  const childRef = (child as { ref?: React.Ref<HTMLElement> }).ref;

  return React.cloneElement(child, {
    ...props,
    ...child.props,
    className: cn(className, child.props.className as string | undefined),
    ref: mergeRefs(forwardedRef, childRef),
    onClick: composeHandlers(
      props.onClick as ((event: React.MouseEvent<HTMLElement>) => void) | undefined,
      child.props.onClick as ((event: React.MouseEvent<HTMLElement>) => void) | undefined,
    ),
    onKeyDown: composeHandlers(
      props.onKeyDown as ((event: React.KeyboardEvent<HTMLElement>) => void) | undefined,
      child.props.onKeyDown as ((event: React.KeyboardEvent<HTMLElement>) => void) | undefined,
    ),
  });
});
