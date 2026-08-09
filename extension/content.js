(function () {
  var pathname = window.location.pathname;

  if (pathname.indexOf('/read/') === -1) return;

  chrome.storage.sync.get(
    {
      serverUrl: '',
      obsidianEnabled: false,
      obsidianVault: '',
      obsidianFolder: 'Readflow',
      obsidianPort: 27124,
    },
    function (settings) {
      if (!settings.obsidianEnabled || !settings.obsidianVault) return;
      if (!settings.serverUrl) return;

      try {
        if (window.location.origin !== new URL(settings.serverUrl).origin) return;
      } catch (e) {
        return;
      }

      injectButton();
    }
  );

  function injectButton() {
    var btn = document.createElement('button');
    btn.className = 'rf-ob-clip-btn';
    btn.title = 'Clip to Obsidian';
    btn.innerHTML =
      '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>';
    document.body.appendChild(btn);

    btn.addEventListener('click', function () {
      var articleId = extractArticleId();
      if (!articleId) return;

      btn.classList.add('clipping');
      showToast('Clipping to Obsidian...');

      chrome.runtime.sendMessage(
        { action: 'clipToObsidian', articleId: articleId },
        function (resp) {
          btn.classList.remove('clipping');
          if (chrome.runtime.lastError) {
            showToast(chrome.runtime.lastError.message, true);
            return;
          }
          if (resp && resp.success) {
            showToast('Clipped to Obsidian!', false);
          } else {
            showToast(resp && resp.error ? resp.error : 'Failed to clip', true);
            btn.classList.add('error');
            setTimeout(function () { btn.classList.remove('error'); }, 2000);
          }
        }
      );
    });
  }

  function extractArticleId() {
    var idx = window.location.pathname.indexOf('/read/');
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
