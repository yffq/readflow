(function (root) {
  'use strict';

  async function openInCurrentTab(chromeApi, uri) {
    if (!uri || uri.indexOf('obsidian://') !== 0) {
      throw new Error('Invalid Obsidian URL.');
    }
    var tabs = await chromeApi.tabs.query({ active: true, currentWindow: true });
    if (!tabs || !tabs[0] || !tabs[0].id) {
      throw new Error('No active browser tab found.');
    }
    await chromeApi.tabs.update(tabs[0].id, { url: uri });
  }

  function buildNewNoteUri(vault, filePath, content, useClipboard) {
    var params = ['file=' + encodeURIComponent(filePath)];
    if (vault) params.push('vault=' + encodeURIComponent(vault));
    if (useClipboard) {
      params.push('clipboard');
      params.push('content=' + encodeURIComponent('Readflow could not access the clipboard. Please try saving again.'));
    } else {
      params.push('content=' + encodeURIComponent(content));
    }
    return 'obsidian://new?' + params.join('&');
  }

  var api = { openInCurrentTab: openInCurrentTab, buildNewNoteUri: buildNewNoteUri };
  root.ObsidianNavigation = api;
  if (typeof module !== 'undefined' && module.exports) module.exports = api;
})(typeof globalThis !== 'undefined' ? globalThis : this);
