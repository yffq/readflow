const serverUrlInput = document.getElementById('serverUrl');
const apiKeyInput = document.getElementById('apiKey');
const saveBtn = document.getElementById('save-btn');

chrome.storage.sync.get({ apiKey: '', serverUrl: '' }, (settings) => {
  serverUrlInput.value = settings.serverUrl;
  apiKeyInput.value = settings.apiKey;
});

saveBtn.addEventListener('click', () => {
  chrome.storage.sync.set({
    serverUrl: serverUrlInput.value.trim(),
    apiKey: apiKeyInput.value.trim()
  }, () => {
    saveBtn.textContent = 'Saved!';
    setTimeout(() => { saveBtn.textContent = 'Save'; }, 1500);
  });
});
