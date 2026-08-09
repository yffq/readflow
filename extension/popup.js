var infoEl = document.getElementById('page-info');
var saveBtn = document.getElementById('save-btn');
var saveStatusEl = document.getElementById('save-status');
var obsidianSection = document.getElementById('obsidian-section');
var clipBtn = document.getElementById('clip-btn');
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
    { obsidianEnabled: false, obsidianVault: '' },
    function (settings) {
      if (settings.obsidianEnabled) {
        obsidianSection.style.display = 'block';
      }
    }
  );
}

saveBtn.addEventListener('click', function () {
  if (!currentTab) return;
  saveBtn.disabled = true;
  saveStatusEl.textContent = 'Saving...';
  saveStatusEl.className = 'status';

  chrome.storage.sync.get({ apiKey: '', serverUrl: '' }, function (settings) {
    if (!settings.apiKey) {
      saveStatusEl.textContent = 'Configure API key in Settings.';
      saveStatusEl.className = 'status status-err';
      saveBtn.disabled = false;
      return;
    }

    var apiUrl = settings.serverUrl.replace(/\/$/, '') + '/api/v1/save';
    var body = {
      url: currentTab.url,
      title: currentTab.title || ''
    };

    fetch(apiUrl, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer ' + settings.apiKey
      },
      body: JSON.stringify(body)
    })
      .then(function (res) {
        if (!res.ok) {
          return res.json().then(function (d) { throw new Error(d.error || 'HTTP ' + res.status); });
        }
        return res.json();
      })
      .then(function (data) {
        if (data.error) {
          saveStatusEl.textContent = 'Error: ' + data.error;
          saveStatusEl.className = 'status status-err';
        } else {
          saveStatusEl.textContent = 'Saved!';
          saveStatusEl.className = 'status status-ok';
        }
        saveBtn.disabled = false;
      })
      .catch(function (err) {
        saveStatusEl.textContent = 'Failed: ' + err.message;
        saveStatusEl.className = 'status status-err';
        saveBtn.disabled = false;
      });
  });
});

clipBtn.addEventListener('click', function () {
  if (!currentTab) return;
  var articleId = extractArticleId(currentTab.url);
  if (!articleId) return;

  clipBtn.disabled = true;
  clipStatusEl.textContent = 'Clipping...';
  clipStatusEl.className = 'status';

  chrome.runtime.sendMessage(
    { action: 'clipToObsidian', articleId: articleId },
    function (resp) {
      clipBtn.disabled = false;
      if (chrome.runtime.lastError) {
        clipStatusEl.textContent = chrome.runtime.lastError.message;
        clipStatusEl.className = 'status status-err';
        return;
      }
      if (resp && resp.success) {
        if (resp.useClipboard) {
          navigator.clipboard.writeText(resp.markdown).then(function () {
            clipStatusEl.textContent = 'Article too long for URI. Copied to clipboard.';
            clipStatusEl.className = 'status status-ok';
          }).catch(function () {
            clipStatusEl.textContent = 'Failed to copy to clipboard.';
            clipStatusEl.className = 'status status-err';
          });
        } else {
          clipStatusEl.textContent = 'Clipped to Obsidian!';
          clipStatusEl.className = 'status status-ok';
        }
      } else {
        clipStatusEl.textContent = (resp && resp.error) ? resp.error : 'Failed to clip.';
        clipStatusEl.className = 'status status-err';
      }
    }
  );
});

document.getElementById('open-options').addEventListener('click', function (e) {
  e.preventDefault();
  chrome.runtime.openOptionsPage();
});
