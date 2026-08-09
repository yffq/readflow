(function () {
  var pathname = window.location.pathname;

  if (pathname.lastIndexOf('/read/') === -1) return;

  chrome.storage.sync.get(
    {
      serverUrl: '',
      obsidianEnabled: false,
      obsidianFolders: ['Readflow'],
      obsidianLastFolder: 'Readflow',
    },
    function (settings) {
      if (!settings.obsidianEnabled) return;
      if (!settings.serverUrl) return;

      try {
        if (window.location.origin !== new URL(settings.serverUrl).origin) return;
      } catch (e) {
        return;
      }

      injectButtons(settings.obsidianFolders, settings.obsidianLastFolder);
    }
  );

  function injectButtons(folders, lastFolder) {
    var wrapper = document.createElement('div');
    wrapper.className = 'rf-ob-wrapper';
    document.body.appendChild(wrapper);

    for (var i = 0; i < folders.length; i++) {
      var btn = document.createElement('div');
      btn.className = 'rf-ob-clip-btn';
      btn.textContent = folders[i];
      btn.title = 'Clip to Obsidian — ' + folders[i];
      if (folders[i] === lastFolder) {
        btn.classList.add('active');
      }
      btn.addEventListener('click', (function (folder, el) {
        return function () {
          clipArticle(folder, el);
        };
      })(folders[i], btn));
      wrapper.appendChild(btn);
    }
  }

  function clipArticle(folder, btnEl) {
    var articleId = extractArticleId();
    if (!articleId) return;

    btnEl.classList.add('clipping');
    showToast('Clipping to ' + folder + '...');

    chrome.runtime.sendMessage(
      { action: 'clipToObsidian', articleId: articleId, folder: folder },
      function (resp) {
        btnEl.classList.remove('clipping');
        if (chrome.runtime.lastError) {
          showToast(chrome.runtime.lastError.message, true);
          return;
        }
        if (resp && resp.success) {
          if (resp.useClipboard) {
            if (resp.obsidianUri) {
              window.open(resp.obsidianUri, '_blank');
            }
            navigator.clipboard.writeText(resp.markdown).then(function () {
              showToast('Note created, body copied — paste into Obsidian.', false);
            }).catch(function () {
              showToast('Note created, but failed to copy body to clipboard.', true);
            });
          } else if (resp.obsidianUri) {
            window.open(resp.obsidianUri, '_blank');
            showToast('Clipped to ' + folder + '!', false);
          } else {
            showToast('Clipped to ' + folder + '!', false);
          }
          updateActiveButton(folder);
        } else {
          showToast(resp && resp.error ? resp.error : 'Failed to clip', true);
          btnEl.classList.add('error');
          setTimeout(function () { btnEl.classList.remove('error'); }, 2000);
        }
      }
    );
  }

  function updateActiveButton(folder) {
    var btns = document.querySelectorAll('.rf-ob-clip-btn');
    for (var i = 0; i < btns.length; i++) {
      if (btns[i].textContent === folder) {
        btns[i].classList.add('active');
      } else {
        btns[i].classList.remove('active');
      }
    }
  }

  function extractArticleId() {
    var idx = window.location.pathname.lastIndexOf('/read/');
    if (idx === -1) return null;
    var after = window.location.pathname.slice(idx + 6);
    var end = after.indexOf('/');
    return end === -1 ? after : after.slice(0, end);
  }

  function showToast(msg, isError) {
    var existing = document.querySelector('.rf-ob-toast');
    if (existing) existing.remove();

    var toast = document.createElement('div');
    toast.className = 'rf-ob-toast';
    toast.textContent = msg;
    if (isError) toast.classList.add('error');
    else toast.classList.add('success');
    document.body.appendChild(toast);

    requestAnimationFrame(function () { toast.classList.add('show'); });

    setTimeout(function () {
      toast.classList.remove('show');
      setTimeout(function () { toast.remove(); }, 200);
    }, 2500);
  }
})();
