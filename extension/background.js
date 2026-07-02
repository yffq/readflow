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
      console.log('Readflow: Please configure API key and server URL in extension options.');
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
      .then(res => {
        if (!res.ok) {
          return res.json().then(d => { throw new Error(d.error || 'HTTP ' + res.status); });
        }
        return res.json();
      })
      .then(data => {
        if (data.error) {
          console.log('Readflow error: ' + data.error);
        } else {
          console.log('Readflow: Saved!');
        }
      })
      .catch(err => console.log('Readflow: ' + err.message));
  });
});

function loadSettings() {
  return chrome.storage.sync.get({ apiKey: '', serverUrl: '' });
}
