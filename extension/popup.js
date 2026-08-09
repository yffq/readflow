var infoEl = document.getElementById('page-info');
var saveBtn = document.getElementById('save-btn');
var saveStatusEl = document.getElementById('save-status');
var obsidianSection = document.getElementById('obsidian-section');
var folderButtonsEl = document.getElementById('folder-buttons');
var clipStatusEl = document.getElementById('clip-status');

var currentTab = null;

chrome.tabs.query({ active: true, currentWindow: true }, function (tabs) {
  if (tabs[0]) {
    currentTab = tabs[0];
    infoEl.textContent = (currentTab.title || 'Untitled') + '\n' + currentTab.url;
    checkObsidianSection();
  }
});

function isReadflowArticlePage(url) {
  return url && url.lastIndexOf('/read/') !== -1;
}

function extractArticleId(url) {
  var idx = url.lastIndexOf('/read/');
  if (idx === -1) return null;
  var after = url.slice(idx + 6);
  var end = after.indexOf('/');
  if (end !== -1) after = after.slice(0, end);
  end = after.indexOf('?');
  if (end !== -1) after = after.slice(0, end);
  end = after.indexOf('#');
  if (end !== -1) after = after.slice(0, end);
  return after || null;
}

function checkObsidianSection() {
  if (!isReadflowArticlePage(currentTab && currentTab.url)) return;

  chrome.storage.sync.get(
    {
      obsidianEnabled: false,
      obsidianFolders: ['Readflow'],
      obsidianLastFolder: 'Readflow'
    },
    function (settings) {
      if (!settings.obsidianEnabled) return;
      obsidianSection.style.display = 'block';
      renderFolderButtons(settings.obsidianFolders, settings.obsidianLastFolder);
    }
  );
}

function renderFolderButtons(folders, lastFolder) {
  folderButtonsEl.innerHTML = '';
  for (var i = 0; i < folders.length; i++) {
    var btn = document.createElement('button');
    btn.className = 'folder-btn';
    btn.textContent = folders[i];
    if (folders[i] === lastFolder) {
      btn.classList.add('active');
    }
    btn.addEventListener('click', (function (folder) {
      return function () { clipArticle(folder); };
    })(folders[i]));
    folderButtonsEl.appendChild(btn);
  }
}

function clipArticle(folder) {
  if (!currentTab) return;
  var articleId = extractArticleId(currentTab.url);
  if (!articleId) return;

  var btns = folderButtonsEl.querySelectorAll('button');
  for (var i = 0; i < btns.length; i++) {
    btns[i].disabled = true;
  }
  clipStatusEl.textContent = 'Clipping to ' + folder + '...';
  clipStatusEl.className = 'status';

  chrome.runtime.sendMessage(
    { action: 'clipToObsidian', articleId: articleId, folder: folder },
    function (resp) {
      for (var i = 0; i < btns.length; i++) {
        btns[i].disabled = false;
      }
      if (chrome.runtime.lastError) {
        clipStatusEl.textContent = chrome.runtime.lastError.message;
        clipStatusEl.className = 'status status-err';
        return;
      }
      if (resp && resp.success) {
        if (resp.useClipboard) {
          navigator.clipboard.writeText(resp.markdown).then(function () {
            clipStatusEl.textContent = 'Note created, body copied — paste into Obsidian.';
            clipStatusEl.className = 'status status-ok';
          }).catch(function () {
            clipStatusEl.textContent = 'Note created, but failed to copy body.';
            clipStatusEl.className = 'status status-err';
          });
        } else {
          clipStatusEl.textContent = 'Clipped to ' + folder + '!';
          clipStatusEl.className = 'status status-ok';
        }
        updateActiveButton(folder);
      } else {
        clipStatusEl.textContent = (resp && resp.error) ? resp.error : 'Failed to clip.';
        clipStatusEl.className = 'status status-err';
      }
    }
  );
}

function updateActiveButton(folder) {
  var btns = folderButtonsEl.querySelectorAll('button');
  for (var i = 0; i < btns.length; i++) {
    if (btns[i].textContent === folder) {
      btns[i].classList.add('active');
    } else {
      btns[i].classList.remove('active');
    }
  }
}

saveBtn.addEventListener('click', function () {
  if (!currentTab) return;
  saveBtn.disabled = true;
  saveStatusEl.textContent = 'Saving...';
  saveStatusEl.className = 'status';

  chrome.runtime.sendMessage({
    action: 'saveToReadflow',
    page: { url: currentTab.url, title: currentTab.title || '' }
  }, function (result) {
    saveBtn.disabled = false;
    if (chrome.runtime.lastError) {
      saveStatusEl.textContent = 'Failed: ' + chrome.runtime.lastError.message;
      saveStatusEl.className = 'status status-err';
      return;
    }
    if (!result || !result.success) {
      saveStatusEl.textContent = (result && result.error) || 'Failed to save page.';
      saveStatusEl.className = 'status status-err';
      return;
    }
    saveStatusEl.textContent = result.data && result.data.status === 'duplicate' ? 'Already saved.' : 'Saved!';
    saveStatusEl.className = 'status status-ok';
  });
});

document.getElementById('open-options').addEventListener('click', function (e) {
  e.preventDefault();
  chrome.runtime.openOptionsPage();
});
