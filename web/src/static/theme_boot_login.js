(function () {
  try {
    if (localStorage.getItem('aed-admin-theme-light-default-v2') !== '1') {
      localStorage.setItem('aed-admin-theme', 'light');
      localStorage.setItem('aed-admin-theme-light-default-v2', '1');
    }
    var theme = localStorage.getItem('aed-admin-theme');
    document.documentElement.classList.add(theme === 'light' ? 'light' : 'dark');
  } catch (e) {
    document.documentElement.classList.add('light');
  }
})();
