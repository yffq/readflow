chrome.runtime.onInstalled.addListener(() => {
  chrome.contextMenus.create({
    id: 'save-link',
    title: 'Save Link to Readflow',
    contexts: ['link']
  });
  chrome.contextMenus.create({
    id: 'save-page',
    title: 'Save Page to Readflow',
    contexts: ['page']
  });
});

chrome.contextMenus.onClicked.addListener((info, tab) => {
  let url = '';
  let title = '';

  if (info.menuItemId === 'save-link') {
    url = info.linkUrl;
    title = '';
  } else if (info.menuItemId === 'save-page') {
    url = info.pageUrl;
    title = tab.title || '';
  }

  if (!url) return;

  loadSettings().then(settings => {
    if (!settings.apiKey || !settings.serverUrl) {
      notifyUser('Readflow: Please configure API key and server URL in extension options.');
      return;
    }

    const body = { url: url };
    if (title) body.title = title;

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
          notifyUser('Readflow: ' + data.error);
        } else {
          notifyUser('Readflow: Saved!');
        }
      })
      .catch(() => notifyUser('Readflow: Failed to save.'));
  });
});

function loadSettings() {
  return chrome.storage.sync.get({ apiKey: '', serverUrl: '' });
}

function notifyUser(msg) {
  console.log(msg);
}
