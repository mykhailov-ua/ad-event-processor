(function () {
  try {
    if (localStorage.getItem('aed-admin-theme-dark-default-v3') !== '1') {
      localStorage.setItem('aed-admin-theme', 'dark');
      localStorage.setItem('aed-admin-theme-dark-default-v3', '1');
    }
    var theme = localStorage.getItem('aed-admin-theme');
    document.documentElement.classList.add(theme === 'light' ? 'light' : 'dark');
  } catch (e) {
    document.documentElement.classList.add('dark');
  }
})();
