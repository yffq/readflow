(function () {
  const linkMenu = document.getElementById('link-menu');
  const toast = document.getElementById('toast');
  const csrfMeta = document.querySelector('meta[name="csrf-token"]');
  const csrfToken = csrfMeta ? csrfMeta.getAttribute('content') : '';
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

  function resolveLinkUrl(rawUrl) {
    if (!rawUrl) return '';
    var value = rawUrl.trim();
    if (!value || value.startsWith('#') || value.toLowerCase().startsWith('mailto:') || value.toLowerCase().startsWith('javascript:')) {
      return '';
    }
    var base = content ? (content.dataset.baseUrl || window.location.href) : window.location.href;
    try {
      return new URL(value, base).href;
    } catch (_) {
      return value;
    }
  }

  function prepareArticleLinks(root) {
    root.querySelectorAll('a[href]').forEach(function (link) {
      var url = resolveLinkUrl(link.getAttribute('href'));
      if (!url) return;
      link.setAttribute('data-rf-url', url);
      link.setAttribute('role', 'button');
      link.removeAttribute('href');
    });
  }

  function closestArticleLink(node) {
    if (!node || !node.closest) return null;
    return node.closest('a[data-rf-url]');
  }

  function copyText(text) {
    if (navigator.clipboard && navigator.clipboard.writeText) {
      return navigator.clipboard.writeText(text);
    }
    return new Promise(function (resolve, reject) {
      var textarea = document.createElement('textarea');
      textarea.value = text;
      textarea.setAttribute('readonly', '');
      textarea.style.position = 'fixed';
      textarea.style.left = '-9999px';
      document.body.appendChild(textarea);
      textarea.select();
      try {
        if (document.execCommand('copy')) resolve();
        else reject(new Error('copy failed'));
      } catch (err) {
        reject(err);
      } finally {
        document.body.removeChild(textarea);
      }
    });
  }

  function showLinkMenu(x, y, url) {
    if (!linkMenu) return;
    menuTargetUrl = url;
    linkMenu.querySelector('.lm-save').dataset.url = url;
    linkMenu.querySelector('.lm-open').dataset.url = url;
    linkMenu.querySelector('.lm-copy').dataset.url = url;
    linkMenu.classList.remove('hidden');

    var menuWidth = linkMenu.offsetWidth || 220;
    var menuHeight = linkMenu.offsetHeight || 132;
    var left = Math.min(x, window.innerWidth - menuWidth - 10);
    var top = Math.min(y, window.innerHeight - menuHeight - 10);
    left = Math.max(10, left);
    top = Math.max(10, top);

    linkMenu.style.left = left + 'px';
    linkMenu.style.top = top + 'px';
    setTimeout(() => menuTargetUrl = url, 0);
  }

  // Long-press / right-click on links in article content to show menu
  var content = document.querySelector('.article-content');
  var longPressTimer = null;
  var suppressNextLinkClick = false;

  if (content) {
    prepareArticleLinks(content);

    content.addEventListener('contextmenu', function (e) {
      var a = closestArticleLink(e.target);
      if (!a) return;
      var href = a.getAttribute('data-rf-url');
      if (!href) return;
      e.preventDefault();
      showLinkMenu(e.clientX, e.clientY, href);
    });

    content.addEventListener('touchstart', function (e) {
      var a = closestArticleLink(e.target);
      if (!a) return;
      var href = a.getAttribute('data-rf-url');
      if (!href) return;

      var touch = e.touches[0];
      longPressTimer = setTimeout(function () {
        longPressTimer = null;
        suppressNextLinkClick = true;
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

    content.addEventListener('click', function (e) {
      var a = closestArticleLink(e.target);
      if (!a) return;
      var href = a.getAttribute('data-rf-url');
      if (!href) return;
      if (suppressNextLinkClick) {
        suppressNextLinkClick = false;
        e.preventDefault();
        e.stopPropagation();
        return;
      }
      e.preventDefault();
      e.stopPropagation();
      showLinkMenu(e.clientX || window.innerWidth / 2, e.clientY || window.innerHeight / 2, href);
    }, true);
  }

  if (linkMenu) {
    var saveButton = linkMenu.querySelector('.lm-save');
    var saveButtonText = saveButton ? saveButton.textContent : '📥 Save to Read Later';

    function setSaveButtonState(text, disabled) {
      if (!saveButton) return;
      saveButton.textContent = text;
      saveButton.disabled = !!disabled;
    }

    function resetSaveButtonSoon(delay) {
      setTimeout(function () {
        setSaveButtonState(saveButtonText, false);
      }, delay || 1200);
    }

    saveButton.addEventListener('click', function (e) {
      e.preventDefault();
      e.stopPropagation();
      var url = menuTargetUrl;
      if (!url) {
        showToast('No link selected.');
        return;
      }
      if (!csrfToken) {
        var missingTokenMessage = 'Unable to save: missing session token. Please refresh and try again.';
        setSaveButtonState('Failed to save', true);
        showToast(missingTokenMessage);
        resetSaveButtonSoon(1800);
        return;
      }

      setSaveButtonState('Saving...', true);
      showToast('Saving to Read Later...');
      fetch('/save-link', {
        method: 'POST',
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'X-Requested-With': 'XMLHttpRequest'
        },
        body: JSON.stringify({ url: url, csrf_token: csrfToken })
      })
        .then(function (res) {
          var contentType = res.headers.get('content-type') || '';
          if (res.redirected || !contentType.includes('application/json')) {
            throw new Error('Unable to save. Please log in again and retry.');
          }
          return res.json().catch(function () { return {}; }).then(function (data) {
            if (!res.ok) throw new Error(data.error || 'Failed to save.');
            var message = data.status === 'duplicate' ? 'Already in Read Later' : 'Saved to Read Later';
            setSaveButtonState(message, true);
            showToast(message);
            setTimeout(function () {
              hideLinkMenu();
              setSaveButtonState(saveButtonText, false);
            }, 900);
          });
        })
        .catch(function (err) {
          var message = err.message || 'Failed to save.';
          setSaveButtonState('Failed to save', true);
          showToast(message);
          resetSaveButtonSoon(1800);
        });
    });

    linkMenu.querySelector('.lm-open').addEventListener('click', function () {
      var url = menuTargetUrl;
      hideLinkMenu();
      if (url) window.open(url, '_blank', 'noopener');
    });

    linkMenu.querySelector('.lm-copy').addEventListener('click', function () {
      var url = menuTargetUrl;
      hideLinkMenu();
      if (!url) {
        showToast('Failed to copy.');
        return;
      }
      copyText(url)
        .then(function () { showToast('Copied'); })
        .catch(function () { showToast('Failed to copy.'); });
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
