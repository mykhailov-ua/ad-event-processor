import { createRoot } from 'react-dom/client';
import { StandaloneBoot } from './standalone_boot.js';

const rootEl = document.getElementById('root');
if (rootEl) {
  createRoot(rootEl).render(<StandaloneBoot />);
}
