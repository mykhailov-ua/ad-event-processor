const path = window.location.pathname;
const rootEl = document.getElementById('root');

if (!rootEl) {
  // no-op
} else if (path === '/bootstrap' || path === '/install/done') {
  void import('./standalone_mount.js');
} else {
  void import('./main_mount.js');
}
