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

  // Long-press / right-click on links in article content to show menu
  var content = document.querySelector('.article-content');
  var longPressTimer = null;

  if (content) {
    content.addEventListener('contextmenu', function (e) {
      var a = e.target.closest('a[href]');
      if (!a) return;
      var href = a.getAttribute('href');
      if (!href || href.startsWith('#') || href.startsWith('mailto:') || href.startsWith('javascript:')) return;
      e.preventDefault();
      showLinkMenu(e.clientX, e.clientY, href);
    });

    content.addEventListener('touchstart', function (e) {
      var a = e.target.closest('a[href]');
      if (!a) return;
      var href = a.getAttribute('href');
      if (!href || href.startsWith('#') || href.startsWith('mailto:') || href.startsWith('javascript:')) return;

      var touch = e.touches[0];
      longPressTimer = setTimeout(function () {
        longPressTimer = null;
        showLinkMenu(touch.clientX, touch.clientY, href);
      }, 500);
    });

    content.addEventListener('touchend', function () {
      if (longPressTimer) {
        clearTimeout(longPressTimer);
        longPressTimer = null;
      }
    });

    content.addEventListener('touchmove', function () {
      if (longPressTimer) {
        clearTimeout(longPressTimer);
        longPressTimer = null;
      }
    });
  }

  if (linkMenu) {
    linkMenu.querySelector('.lm-save').addEventListener('click', function () {
      var url = menuTargetUrl;
      hideLinkMenu();
      if (!url) return;

      var body = 'url=' + encodeURIComponent(url) + '&csrf_token=' + encodeURIComponent(csrfToken);
      fetch('/save', {
        method: 'POST',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        body: body
      })
        .then(function () { showToast('Saved to Read Later'); })
        .catch(function () { showToast('Failed to save.'); });
    });

    linkMenu.querySelector('.lm-open').addEventListener('click', function () {
      var url = menuTargetUrl;
      hideLinkMenu();
      if (url) window.open(url, '_blank', 'noopener');
    });
  }

  document.addEventListener('click', function (e) {
    if (linkMenu && !linkMenu.classList.contains('hidden')) {
      if (!linkMenu.contains(e.target)) {
        hideLinkMenu();
      }
    }
  });

  document.addEventListener('keydown', function (e) {
    if (e.key === 'Escape') hideLinkMenu();
  });

  // --- Batch selection ---
  var selectAll = document.getElementById('select-all');
  var batchBar = document.getElementById('batch-bar');
  var selectedCount = document.getElementById('selected-count');
  var batchDeleteBtn = document.getElementById('batch-delete-btn');

  function updateBatchUI() {
    var checked = document.querySelectorAll('.article-checkbox[data-article-id]:checked');
    var count = checked.length;
    if (count > 0) {
      batchBar.classList.remove('hidden');
      selectedCount.textContent = count + ' selected';
      batchDeleteBtn.disabled = false;
      if (selectAll) selectAll.indeterminate = false;
    } else {
      batchBar.classList.add('hidden');
      selectedCount.textContent = '0 selected';
      batchDeleteBtn.disabled = true;
    }
  }

  if (selectAll) {
    selectAll.addEventListener('change', function () {
      var checkboxes = document.querySelectorAll('.article-checkbox[data-article-id]');
      checkboxes.forEach(function (cb) { cb.checked = selectAll.checked; });
      updateBatchUI();
    });
  }

  document.querySelectorAll('.article-checkbox[data-article-id]').forEach(function (cb) {
    cb.addEventListener('change', function () {
      var checkboxes = document.querySelectorAll('.article-checkbox[data-article-id]');
      var checked = document.querySelectorAll('.article-checkbox[data-article-id]:checked');
      if (selectAll) {
        selectAll.checked = checked.length === checkboxes.length;
        selectAll.indeterminate = checked.length > 0 && checked.length < checkboxes.length;
      }
      updateBatchUI();
    });
  });

  window.batchDelete = function () {
    var checked = document.querySelectorAll('.article-checkbox[data-article-id]:checked');
    if (checked.length === 0) return;
    if (!confirm('Delete ' + checked.length + ' article(s)? This cannot be undone.')) return;

    var ids = [];
    checked.forEach(function (cb) { ids.push(cb.value); });

    fetch('/delete-batch', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ids: ids, csrf_token: csrfToken })
    })
      .then(function (res) {
        if (res.ok) {
          location.reload();
        } else {
          showToast('Failed to delete articles.');
        }
      })
      .catch(function () { showToast('Failed to delete articles.'); });
  };

  window.cancelSelection = function () {
    document.querySelectorAll('.article-checkbox[data-article-id]').forEach(function (cb) {
      cb.checked = false;
    });
    if (selectAll) selectAll.checked = false;
    updateBatchUI();
  };

  // Clicking article link should not toggle checkbox
  document.querySelectorAll('.article-title').forEach(function (link) {
    link.addEventListener('click', function (e) {
      e.stopPropagation();
    });
  });
})();
