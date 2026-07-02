const infoEl = document.getElementById('page-info');
const saveBtn = document.getElementById('save-btn');
const statusEl = document.getElementById('status');

let currentTab = null;

chrome.tabs.query({ active: true, currentWindow: true }, (tabs) => {
  if (tabs[0]) {
    currentTab = tabs[0];
    infoEl.textContent = (currentTab.title || 'Untitled') + '\n' + currentTab.url;
  }
});

saveBtn.addEventListener('click', () => {
  if (!currentTab) return;
  saveBtn.disabled = true;
  statusEl.textContent = 'Saving...';
  statusEl.className = 'status';

  chrome.storage.sync.get({ apiKey: '', serverUrl: '' }, (settings) => {
    if (!settings.apiKey) {
      statusEl.textContent = 'Configure API key in Settings.';
      statusEl.className = 'status status-err';
      saveBtn.disabled = false;
      return;
    }

    const body = {
      url: currentTab.url,
      title: currentTab.title || ''
    };

    fetch(settings.serverUrl.replace(/\/$/, '') + '/api/v1/save', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer ' + settings.apiKey
      },
      body: JSON.stringify(body)
    })
      .then(res => res.json())
      .then(data => {
        if (data.error) {
          statusEl.textContent = 'Error: ' + data.error;
          statusEl.className = 'status status-err';
        } else {
          statusEl.textContent = 'Saved!';
          statusEl.className = 'status status-ok';
        }
        saveBtn.disabled = false;
      })
      .catch(() => {
        statusEl.textContent = 'Failed to save.';
        statusEl.className = 'status status-err';
        saveBtn.disabled = false;
      });
  });
});

document.getElementById('open-options').addEventListener('click', (e) => {
  e.preventDefault();
  chrome.runtime.openOptionsPage();
});
