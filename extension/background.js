importScripts('readflow-api.js', 'obsidian-navigation.js');

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

chrome.contextMenus.onClicked.addListener(async (info, tab) => {
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
  const result = await saveToReadflow({ url: url, title: title });
  showSaveNotification(result);
});

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (message.action === 'saveToReadflow') {
    saveToReadflow(message.page).then(sendResponse);
    return true;
  }
  if (message.action === 'clipToObsidian') {
    clipToObsidian(message.articleId, message.folder).then(sendResponse);
    return true;
  }
});

async function saveToReadflow(page) {
  try {
    const settings = await loadSettings();
    const data = await ReadflowAPI.savePage(settings, page);
    return { success: true, data: data };
  } catch (err) {
    return { success: false, error: err.message || 'Failed to save page.' };
  }
}

function showSaveNotification(result) {
  const duplicate = result.success && result.data && result.data.status === 'duplicate';
  chrome.notifications.create({
    type: 'basic',
    iconUrl: 'icons/icon128.png',
    title: result.success ? (duplicate ? 'Already in Readflow' : 'Saved to Readflow') : 'Readflow save failed',
    message: result.success ? (duplicate ? 'This page was already saved.' : 'The page was saved successfully.') : result.error
  });
}

async function clipToObsidian(articleId, folder) {
  try {
    const settings = await loadSettings();
    if (!settings.apiKey || !settings.serverUrl) {
      return { success: false, error: 'Readflow API not configured. Set Server URL and API Key in extension options.' };
    }
    if (!settings.obsidianEnabled) {
      return { success: false, error: 'Obsidian clipping not configured. Enable it in extension options.' };
    }

    folder = folder || (settings.obsidianFolders && settings.obsidianFolders[0]) || 'Readflow';

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
    const filePath = folder.replace(/\/$/, '') + '/' + filename;

    const copied = await copyMarkdownToClipboard(markdown);
    const obsidianUri = ObsidianNavigation.buildNewNoteUri(settings.obsidianVault, filePath, markdown, copied);
    await openObsidianTab(obsidianUri);
    await chrome.storage.sync.set({ obsidianLastFolder: folder });
    return { success: true };
  } catch (err) {
    return { success: false, error: err.message || 'Unknown error.' };
  }
}

function formatArticleMarkdown(article, lite) {
  lite = lite || false;
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

  if (lite) {
    lines.push('<!-- Paste full content here (Cmd+V / Ctrl+V) -->');
  } else if (article.content_markdown) {
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

async function openObsidianTab(uri) {
  await ObsidianNavigation.openInCurrentTab(chrome, uri);
}

async function copyMarkdownToClipboard(markdown) {
  try {
    const tabs = await chrome.tabs.query({ active: true, currentWindow: true });
    if (!tabs || !tabs[0] || !tabs[0].id) return false;
    const response = await chrome.tabs.sendMessage(tabs[0].id, {
      action: 'copyReadflowMarkdown',
      text: markdown
    });
    return Boolean(response && response.success);
  } catch (err) {
    return false;
  }
}

function loadSettings() {
  return chrome.storage.sync.get({
    apiKey: '',
    serverUrl: '',
    obsidianEnabled: false,
    obsidianVault: '',
    obsidianFolders: ['Readflow'],
    obsidianLastFolder: 'Readflow'
  });
}
