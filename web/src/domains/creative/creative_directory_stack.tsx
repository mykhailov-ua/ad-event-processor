import type { ReactNode } from 'react';

import { CreativeNav } from '@/domains/creative/creative_nav';
import { DirectoryStack } from '@/shell/ui_bands';

export type CreativeDirectoryStackProps = {
  children?: ReactNode;
};

export function CreativeDirectoryStack({ children }: CreativeDirectoryStackProps) {
  return (
    <DirectoryStack>
      <CreativeNav />
      {children}
    </DirectoryStack>
  );
}
