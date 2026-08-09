(function (root) {
  'use strict';

  function validateSettings(settings) {
    if (!settings || !String(settings.serverUrl || '').trim()) {
      throw new Error('Configure Server URL in Settings.');
    }
    if (!String(settings.apiKey || '').trim()) {
      throw new Error('Configure API key in Settings.');
    }

    var parsed;
    try {
      parsed = new URL(String(settings.serverUrl).trim());
    } catch (err) {
      throw new Error('Server URL is invalid. Include http:// or https://.');
    }
    if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') {
      throw new Error('Server URL must use http:// or https://.');
    }
    return {
      serverUrl: parsed.href.replace(/\/$/, ''),
      apiKey: String(settings.apiKey).trim()
    };
  }

  function validatePageUrl(rawUrl) {
    var parsed;
    try {
      parsed = new URL(String(rawUrl || ''));
    } catch (err) {
      throw new Error('This page cannot be saved because its URL is invalid.');
    }
    if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') {
      throw new Error('Only http:// and https:// pages can be saved.');
    }
    return parsed.href;
  }

  async function readResponse(response) {
    var text = await response.text();
    var data = null;
    if (text) {
      try { data = JSON.parse(text); } catch (err) { data = null; }
    }
    if (!response.ok) {
      var message = data && data.error ? data.error : ('HTTP ' + response.status);
      if (response.status === 401) message = data && data.error ? data.error : 'Invalid API key.';
      throw new Error(message);
    }
    if (!data) throw new Error('Readflow returned an invalid response.');
    if (data.error) throw new Error(data.error);
    return data;
  }

  async function savePage(settings, page, fetchImpl) {
    var config = validateSettings(settings);
    var url = validatePageUrl(page && page.url);
    var request = fetchImpl || fetch;
    var response;
    try {
      response = await request(config.serverUrl + '/api/v1/save', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': 'Bearer ' + config.apiKey
        },
        body: JSON.stringify({ url: url, title: String(page.title || '') })
      });
    } catch (err) {
      throw new Error('Cannot reach the Readflow server. Check Server URL and network connection.');
    }
    return readResponse(response);
  }

  var api = { validateSettings: validateSettings, validatePageUrl: validatePageUrl, savePage: savePage };
  root.ReadflowAPI = api;
  if (typeof module !== 'undefined' && module.exports) module.exports = api;
})(typeof globalThis !== 'undefined' ? globalThis : this);
