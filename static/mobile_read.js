(function () {
  var menu = document.getElementById('link-menu');
  var toast = document.getElementById('toast');
  var apiKeyMeta = document.querySelector('meta[name="api-key"]');
  var menuUrl = '';
  var apiKey = apiKeyMeta ? apiKeyMeta.getAttribute('content') : '';
  if (apiKey && apiKey.toLowerCase().indexOf('bearer ') === 0) {
    apiKey = apiKey.slice(7).trim();
  }
  var content = document.querySelector('.article-content');

  function showToast(msg) {
    if (!toast) return;
    toast.textContent = msg;
    toast.classList.add('show');
    clearTimeout(toast._t);
    toast._t = setTimeout(function () { toast.classList.remove('show'); }, 2500);
  }

  function hideMenu() {
    if (menu) menu.classList.remove('show');
  }

  function resolveLinkUrl(rawUrl) {
    if (!rawUrl) return '';
    var value = rawUrl.trim();
    var lower = value.toLowerCase();
    if (!value || value.charAt(0) === '#' || lower.indexOf('mailto:') === 0 || lower.indexOf('javascript:') === 0) return '';
    var base = content ? (content.dataset.baseUrl || window.location.href) : window.location.href;
    try {
      return new URL(value, base).href;
    } catch (_) {
      return absolutizeURL(value, base);
    }
  }

  function absolutizeURL(value, base) {
    if (/^https?:\/\//i.test(value)) return value;
    if (value.indexOf('//') === 0) return window.location.protocol + value;
    var originMatch = String(base || '').match(/^(https?:\/\/[^/]+)/i);
    var origin = originMatch ? originMatch[1] : window.location.origin;
    if (value.charAt(0) === '/') return origin + value;
    return String(base || window.location.href).replace(/[?#].*$/, '').replace(/\/[^/]*$/, '/') + value;
  }

  function prepareLinks(root) {
    var links = root.querySelectorAll('a[href]');
    for (var i = 0; i < links.length; i += 1) {
      var link = links[i];
      var url = resolveLinkUrl(link.getAttribute('href'));
      if (!url) continue;
      link.setAttribute('data-rf-url', url);
      link.setAttribute('role', 'button');
      link.removeAttribute('href');
    }
  }

  function closestLink(node) {
    while (node && node !== document) {
      if (node.tagName && node.tagName.toLowerCase() === 'a' && node.getAttribute('data-rf-url')) return node;
      node = node.parentNode;
    }
    return null;
  }

  function postJSON(url, body, headers) {
    if (window.fetch) {
      return fetch(url, {
        method: 'POST',
        headers: headers,
        body: JSON.stringify(body)
      }).then(function (res) {
        return res.json().catch(function () { return {}; }).then(function (data) {
          return { ok: res.ok, status: res.status, data: data };
        });
      });
    }
    return new Promise(function (resolve, reject) {
      var xhr = new XMLHttpRequest();
      xhr.open('POST', url, true);
      Object.keys(headers).forEach(function (key) {
        xhr.setRequestHeader(key, headers[key]);
      });
      xhr.onreadystatechange = function () {
        if (xhr.readyState !== 4) return;
        var data = {};
        try {
          data = JSON.parse(xhr.responseText || '{}');
        } catch (_) {}
        resolve({ ok: xhr.status >= 200 && xhr.status < 300, status: xhr.status, data: data });
      };
      xhr.onerror = function () { reject(new Error('Network error')); };
      xhr.send(JSON.stringify(body));
    });
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

  function showMenu(x, y, url) {
    if (!menu) return;
    menuUrl = url;
    menu.classList.add('show');
    var menuWidth = menu.offsetWidth || 200;
    var menuHeight = menu.offsetHeight || 132;
    menu.style.left = Math.max(10, Math.min(x, window.innerWidth - menuWidth - 10)) + 'px';
    menu.style.top = Math.max(10, Math.min(y, window.innerHeight - menuHeight - 10)) + 'px';
  }

  if (content) {
    prepareLinks(content);

    var timer = null;
    var suppressNextLinkClick = false;
    var lastTouch = null;
    content.addEventListener('contextmenu', function (e) {
      var a = closestLink(e.target);
      var href = a ? a.getAttribute('data-rf-url') : '';
      if (!href) return;
      e.preventDefault();
      showMenu(e.clientX, e.clientY, href);
    });
    content.addEventListener('touchstart', function (e) {
      var a = closestLink(e.target);
      var href = a ? a.getAttribute('data-rf-url') : '';
      if (!href) return;
      var t = e.touches[0];
      if (!t) return;
      lastTouch = { x: t.clientX, y: t.clientY };
      timer = setTimeout(function () {
        suppressNextLinkClick = true;
        showMenu(t.clientX, t.clientY, href);
      }, 500);
    }, { passive: true });
    content.addEventListener('touchend', function () {
      if (timer) {
        clearTimeout(timer);
        timer = null;
      }
    });
    content.addEventListener('touchmove', function () {
      if (timer) {
        clearTimeout(timer);
        timer = null;
      }
    });
    content.addEventListener('click', function (e) {
      var a = closestLink(e.target);
      if (!a) return;
      var href = a.getAttribute('data-rf-url');
      if (!href) return;
      if (suppressNextLinkClick) {
        suppressNextLinkClick = false;
        e.preventDefault();
        e.stopPropagation();
        return;
      }
      suppressNextLinkClick = false;
      e.preventDefault();
      e.stopPropagation();
      var x = lastTouch ? lastTouch.x : e.clientX;
      var y = lastTouch ? lastTouch.y : e.clientY;
      showMenu(x || window.innerWidth / 2, y || window.innerHeight / 2, href);
    }, true);
  }

  if (menu) {
    menu.querySelector('.lm-save').addEventListener('click', function () {
      hideMenu();
      if (!menuUrl) {
        showToast('No link selected');
        return;
      }
      if (!apiKey) {
        showToast('No API key configured');
        return;
      }
      showToast('Saving to Read Later...');
      postJSON(location.origin + '/api/v1/save', { url: menuUrl }, {
        'Accept': 'application/json',
        'Content-Type': 'application/json',
        'Authorization': 'Bearer ' + apiKey
      }).then(function (res) {
        if (!res.ok) throw new Error((res.data && res.data.error) || ('Failed to save: ' + res.status));
        showToast(res.data && res.data.status === 'duplicate' ? 'Already in Read Later' : 'Saved to Read Later');
      }).catch(function (err) {
        showToast(err.message || 'Failed to save.');
      });
    });

    menu.querySelector('.lm-open').addEventListener('click', function () {
      hideMenu();
      if (!menuUrl) return;
      var a = document.createElement('a');
      a.href = menuUrl;
      a.target = '_blank';
      a.rel = 'noopener';
      a.click();
    });

    menu.querySelector('.lm-copy').addEventListener('click', function () {
      hideMenu();
      if (!menuUrl) {
        showToast('Failed to copy.');
        return;
      }
      copyText(menuUrl)
        .then(function () { showToast('Copied'); })
        .catch(function () { showToast('Failed to copy.'); });
    });
  }

  document.addEventListener('click', function (e) {
    if (menu && !menu.contains(e.target)) hideMenu();
  });
})();
