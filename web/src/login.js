import { bootLogin } from './lib/boot.js';

const root = document.getElementById('root');
if (root) bootLogin(root);
