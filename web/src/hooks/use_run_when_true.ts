import { useEffect } from 'react';

export function useRunWhenTrue(signal: boolean, run: () => void) {
  useEffect(() => {
    if (signal) {
      run();
    }
  }, [run, signal]);
}
