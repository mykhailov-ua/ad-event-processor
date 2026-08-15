import { createRoot } from 'react-dom/client';
import { AppBoot } from './app_boot.js';

const rootEl = document.getElementById('root');
if (rootEl) {
  createRoot(rootEl).render(<AppBoot />);
}
