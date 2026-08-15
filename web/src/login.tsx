import { createRoot } from 'react-dom/client';
import { LoginBoot } from './react/login_boot.js';

const root = document.getElementById('root');
if (root) {
  createRoot(root).render(<LoginBoot />);
}
