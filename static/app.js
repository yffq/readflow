(function () {
  const linkMenu = document.getElementById('link-menu');
  const toast = document.getElementById('toast');
  let menuTargetUrl = '';

  function showToast(msg) {
    if (!toast) return;
    toast.textContent = msg;
    toast.classList.remove('hidden');
    clearTimeout(toast._timeout);
    toast._timeout = setTimeout(() => toast.classList.add('hidden'), 2500);
  }

  function hideLinkMenu() {
    if (linkMenu) linkMenu.classList.add('hidden');
  }

  function showLinkMenu(x, y, url) {
    if (!linkMenu) return;
    menuTargetUrl = url;
    linkMenu.querySelector('.lm-save').dataset.url = url;
    linkMenu.querySelector('.lm-open').dataset.url = url;

    var menuWidth = 220;
    var menuHeight = 88;
    var left = Math.min(x, window.innerWidth - menuWidth - 10);
    var top = Math.min(y, window.innerHeight - menuHeight - 10);
    left = Math.max(10, left);
    top = Math.max(10, top);

    linkMenu.style.left = left + 'px';
    linkMenu.style.top = top + 'px';
    linkMenu.classList.remove('hidden');
    setTimeout(() => menuTargetUrl = url, 0);
  }

  // Attach to links inside article-content
  var content = document.querySelector('.article-content');
  if (content) {
    content.addEventListener('click', function (e) {
      var a = e.target.closest('a[href]');
      if (!a) return;
      var href = a.getAttribute('href');
      if (!href || href.startsWith('#') || href.startsWith('mailto:') || href.startsWith('javascript:')) return;

      e.preventDefault();
      showLinkMenu(e.clientX, e.clientY, href);
    });
  }

  // Menu button handlers
  if (linkMenu) {
    linkMenu.querySelector('.lm-save').addEventListener('click', function () {
      var url = menuTargetUrl;
      hideLinkMenu();
      if (!url) return;

      var apiKey = getApiKey();
      if (!apiKey) {
        showToast('Set up an API key in Settings first.');
        return;
      }

      fetch('/api/v1/save', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': 'Bearer ' + apiKey
        },
        body: JSON.stringify({ url: url })
      })
        .then(res => res.json())
        .then(data => {
          if (data.error) {
            showToast('Error: ' + data.error);
          } else {
            showToast('✓ Saved to Read Later');
          }
        })
        .catch(() => showToast('Failed to save. Check your connection.'));
    });

    linkMenu.querySelector('.lm-open').addEventListener('click', function () {
      var url = menuTargetUrl;
      hideLinkMenu();
      if (url) window.open(url, '_blank', 'noopener');
    });
  }

  // Close menu on outside click
  document.addEventListener('click', function (e) {
    if (linkMenu && !linkMenu.classList.contains('hidden')) {
      if (!linkMenu.contains(e.target) && !e.target.closest('.article-content a')) {
        hideLinkMenu();
      }
    }
  });

  // Escape key closes menu
  document.addEventListener('keydown', function (e) {
    if (e.key === 'Escape') hideLinkMenu();
  });

  function getApiKey() {
    // Try to find a stored key. For now, prompt or check sessionStorage.
    var key = sessionStorage.getItem('readflow_api_key');
    if (!key) {
      key = prompt('Enter your Readflow API key (from Settings page):');
      if (key) sessionStorage.setItem('readflow_api_key', key);
    }
    return key;
  }
})();
