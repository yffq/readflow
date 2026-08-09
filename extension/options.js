var serverUrlInput = document.getElementById('serverUrl');
var apiKeyInput = document.getElementById('apiKey');
var obsidianEnabledCheckbox = document.getElementById('obsidianEnabled');
var obsidianVaultInput = document.getElementById('obsidianVault');
var folderListEl = document.getElementById('folder-list');
var addFolderBtn = document.getElementById('add-folder-btn');
var saveBtn = document.getElementById('save-btn');
var saveStatusEl = document.getElementById('save-status');

chrome.storage.sync.get({
  apiKey: '',
  serverUrl: '',
  obsidianEnabled: false,
  obsidianVault: '',
  obsidianFolders: ['Readflow']
}, function (settings) {
  serverUrlInput.value = settings.serverUrl;
  apiKeyInput.value = settings.apiKey;
  obsidianEnabledCheckbox.checked = settings.obsidianEnabled;
  obsidianVaultInput.value = settings.obsidianVault;
  renderFolderList(settings.obsidianFolders || ['Readflow']);
});

function renderFolderList(folders) {
  folderListEl.innerHTML = '';
  for (var i = 0; i < folders.length; i++) {
    var row = document.createElement('div');
    row.className = 'folder-row';

    var input = document.createElement('input');
    input.type = 'text';
    input.value = folders[i];
    input.placeholder = 'Folder name';

    var delBtn = document.createElement('button');
    delBtn.textContent = '\u2715';
    delBtn.title = 'Remove folder';
    if (folders.length <= 1) delBtn.disabled = true;
    delBtn.addEventListener('click', function () {
      var rows = folderListEl.querySelectorAll('.folder-row');
      if (rows.length <= 1) return;
      row.remove();
    });

    row.appendChild(input);
    row.appendChild(delBtn);
    folderListEl.appendChild(row);
  }
}

addFolderBtn.addEventListener('click', function () {
  var row = document.createElement('div');
  row.className = 'folder-row';

  var input = document.createElement('input');
  input.type = 'text';
  input.placeholder = 'Folder name';

  var delBtn = document.createElement('button');
  delBtn.textContent = '\u2715';
  delBtn.title = 'Remove folder';
  delBtn.addEventListener('click', function () {
    var rows = folderListEl.querySelectorAll('.folder-row');
    if (rows.length <= 1) return;
    row.remove();
  });

  row.appendChild(input);
  row.appendChild(delBtn);
  folderListEl.appendChild(row);

  input.focus();
});

saveBtn.addEventListener('click', function () {
  var serverUrl = serverUrlInput.value.trim();
  var apiKey = apiKeyInput.value.trim();
  try {
    var parsedServerUrl = new URL(serverUrl);
    if (parsedServerUrl.protocol !== 'http:' && parsedServerUrl.protocol !== 'https:') throw new Error();
  } catch (err) {
    saveStatusEl.textContent = 'Enter a valid Server URL including http:// or https://.';
    saveStatusEl.className = 'status status-err';
    return;
  }
  if (!apiKey) {
    saveStatusEl.textContent = 'Enter an API key.';
    saveStatusEl.className = 'status status-err';
    return;
  }
  var inputs = folderListEl.querySelectorAll('input');
  var folders = [];
  for (var i = 0; i < inputs.length; i++) {
    var v = inputs[i].value.trim();
    if (v) folders.push(v);
  }
  if (folders.length === 0) folders = ['Readflow'];

  chrome.storage.sync.get({ obsidianLastFolder: '' }, function (s) {
    var lastFolder = s.obsidianLastFolder;
    if (lastFolder && folders.indexOf(lastFolder) === -1) {
      lastFolder = folders[0];
    }

    chrome.storage.sync.set({
      serverUrl: serverUrl.replace(/\/$/, ''),
      apiKey: apiKey,
      obsidianEnabled: obsidianEnabledCheckbox.checked,
      obsidianVault: obsidianVaultInput.value.trim(),
      obsidianFolders: folders,
      obsidianLastFolder: lastFolder || folders[0]
    }, function () {
      if (chrome.runtime.lastError) {
        saveStatusEl.textContent = 'Failed to save settings: ' + chrome.runtime.lastError.message;
        saveStatusEl.className = 'status status-err';
        return;
      }
      saveBtn.textContent = 'Saved!';
      saveStatusEl.textContent = 'Settings saved.';
      saveStatusEl.className = 'status status-ok';
      setTimeout(function () { saveBtn.textContent = 'Save'; }, 1500);
    });
  });
});
