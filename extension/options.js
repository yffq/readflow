const serverUrlInput = document.getElementById('serverUrl');
const apiKeyInput = document.getElementById('apiKey');
const obsidianEnabledCheckbox = document.getElementById('obsidianEnabled');
const obsidianVaultInput = document.getElementById('obsidianVault');
const obsidianFolderInput = document.getElementById('obsidianFolder');
const obsidianPortInput = document.getElementById('obsidianPort');
const saveBtn = document.getElementById('save-btn');

chrome.storage.sync.get({
  apiKey: '',
  serverUrl: '',
  obsidianEnabled: false,
  obsidianVault: '',
  obsidianFolder: 'Readflow',
  obsidianPort: 27124
}, (settings) => {
  serverUrlInput.value = settings.serverUrl;
  apiKeyInput.value = settings.apiKey;
  obsidianEnabledCheckbox.checked = settings.obsidianEnabled;
  obsidianVaultInput.value = settings.obsidianVault;
  obsidianFolderInput.value = settings.obsidianFolder;
  obsidianPortInput.value = settings.obsidianPort;
});

saveBtn.addEventListener('click', () => {
  chrome.storage.sync.set({
    serverUrl: serverUrlInput.value.trim(),
    apiKey: apiKeyInput.value.trim(),
    obsidianEnabled: obsidianEnabledCheckbox.checked,
    obsidianVault: obsidianVaultInput.value.trim(),
    obsidianFolder: obsidianFolderInput.value.trim() || 'Readflow',
    obsidianPort: parseInt(obsidianPortInput.value, 10) || 27124
  }, () => {
    saveBtn.textContent = 'Saved!';
    setTimeout(() => { saveBtn.textContent = 'Save'; }, 1500);
  });
});
