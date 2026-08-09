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

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (message.action === 'clipToObsidian') {
    clipToObsidian(message.articleId).then(sendResponse);
    return true;
  }
});

async function clipToObsidian(articleId) {
  try {
    const settings = await loadSettings();
    if (!settings.apiKey || !settings.serverUrl) {
      return { success: false, error: 'Readflow API not configured. Set Server URL and API Key in extension options.' };
    }
    if (!settings.obsidianEnabled || !settings.obsidianVault) {
      return { success: false, error: 'Obsidian clipping not configured. Enable it in extension options.' };
    }

    const articleUrl = settings.serverUrl.replace(/\/$/, '') + '/api/v1/article/' + encodeURIComponent(articleId);
    const resp = await fetch(articleUrl, {
      headers: { 'Authorization': 'Bearer ' + settings.apiKey }
    });
    if (!resp.ok) {
      if (resp.status === 404) return { success: false, error: 'Article not found.' };
      return { success: false, error: 'Failed to fetch article (HTTP ' + resp.status + ').' };
    }

    let article;
    try {
      article = await resp.json();
    } catch (e) {
      return { success: false, error: 'Invalid response from readflow server.' };
    }

    if (!article || !article.title) {
      return { success: false, error: 'No article data returned.' };
    }

    const markdown = formatArticleMarkdown(article);
    const datePrefix = article.created_at ? article.created_at.slice(0, 10) : '';
    const filename = (datePrefix ? datePrefix + ' ' : '') + sanitizeFilename(article.title) + '.md';
    const url = buildObsidianUrl(
      settings.obsidianVault,
      settings.obsidianFolder || 'Readflow',
      filename,
      settings.obsidianPort || 27124
    );

    const obsResp = await fetch(url, {
      method: 'PUT',
      headers: { 'Content-Type': 'text/markdown' },
      body: markdown
    });
    if (!obsResp.ok) {
      if (obsResp.status === 0) {
        return { success: false, error: 'Cannot connect to Obsidian. Ensure the Local REST API plugin is installed and Obsidian is running.' };
      }
      return { success: false, error: 'Obsidian returned HTTP ' + obsResp.status + '.' };
    }

    return { success: true };
  } catch (err) {
    return { success: false, error: err.message || 'Unknown error.' };
  }
}

function formatArticleMarkdown(article) {
  const lines = [];
  lines.push('---');
  pushField('title', article.title || 'Untitled');
  if (article.author) pushField('author', article.author);
  if (article.url) pushField('source', article.url);
  if (article.site_name) pushField('site', article.site_name);

  const tags = ['readflow'];
  if (article.site_name) tags.push(slug(article.site_name));
  lines.push('tags: [' + tags.join(', ') + ']');

  if (article.created_at) {
    pushField('date', article.created_at.slice(0, 10));
  }
  lines.push('---');
  lines.push('');
  lines.push('# ' + (article.title || 'Untitled'));
  lines.push('');

  if (article.content_markdown) {
    lines.push(article.content_markdown);
  }

  return lines.join('\n');

  function pushField(name, value) {
    var escaped = String(value)
      .replace(/\\/g, '\\\\')
      .replace(/"/g, '\\"')
      .replace(/\n/g, '\\n')
      .replace(/\r/g, '');
    lines.push(name + ': "' + escaped + '"');
  }
}

function sanitizeFilename(name) {
  return name
    .replace(/[<>:"/\\|?*]/g, '')
    .replace(/\s+/g, ' ')
    .trim()
    .slice(0, 200) || 'untitled';
}

function slug(s) {
  return s.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '');
}

function buildObsidianUrl(vault, folder, filename, port) {
  var vaultEnc = encodeURIComponent(vault);
  var segments = folder.split('/').filter(Boolean).map(function (s) { return encodeURIComponent(s); });
  var fileEnc = encodeURIComponent(filename);
  var path = segments.concat(fileEnc).join('/');
  return 'http://localhost:' + port + '/vault/' + vaultEnc + '/' + path;
}

function loadSettings() {
  return chrome.storage.sync.get({
    apiKey: '',
    serverUrl: '',
    obsidianEnabled: false,
    obsidianVault: '',
    obsidianFolder: 'Readflow',
    obsidianPort: 27124
  });
}
