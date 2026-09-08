import * as React from 'react';

import { adminKit } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';

const Label = React.forwardRef<HTMLLabelElement, React.LabelHTMLAttributes<HTMLLabelElement>>(
  ({ className, ...props }, ref) => (
    <label ref={ref} className={cn(adminKit.fieldLabelClass, className)} {...props} />
  )
);
Label.displayName = 'Label';

export { Label };
