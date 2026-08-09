const serverUrlInput = document.getElementById('serverUrl');
const apiKeyInput = document.getElementById('apiKey');
const obsidianEnabledCheckbox = document.getElementById('obsidianEnabled');
const obsidianVaultInput = document.getElementById('obsidianVault');
const obsidianFolderInput = document.getElementById('obsidianFolder');
const saveBtn = document.getElementById('save-btn');

chrome.storage.sync.get({
  apiKey: '',
  serverUrl: '',
  obsidianEnabled: false,
  obsidianVault: '',
  obsidianFolder: 'Readflow'
}, (settings) => {
  serverUrlInput.value = settings.serverUrl;
  apiKeyInput.value = settings.apiKey;
  obsidianEnabledCheckbox.checked = settings.obsidianEnabled;
  obsidianVaultInput.value = settings.obsidianVault;
  obsidianFolderInput.value = settings.obsidianFolder;
});

saveBtn.addEventListener('click', () => {
  chrome.storage.sync.set({
    serverUrl: serverUrlInput.value.trim(),
    apiKey: apiKeyInput.value.trim(),
    obsidianEnabled: obsidianEnabledCheckbox.checked,
    obsidianVault: obsidianVaultInput.value.trim(),
    obsidianFolder: obsidianFolderInput.value.trim() || 'Readflow'
  }, () => {
    saveBtn.textContent = 'Saved!';
    setTimeout(() => { saveBtn.textContent = 'Save'; }, 1500);
  });
});
