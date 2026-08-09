var serverUrlInput = document.getElementById('serverUrl');
var apiKeyInput = document.getElementById('apiKey');
var obsidianEnabledCheckbox = document.getElementById('obsidianEnabled');
var obsidianVaultInput = document.getElementById('obsidianVault');
var folderListEl = document.getElementById('folder-list');
var addFolderBtn = document.getElementById('add-folder-btn');
var saveBtn = document.getElementById('save-btn');

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
      serverUrl: serverUrlInput.value.trim(),
      apiKey: apiKeyInput.value.trim(),
      obsidianEnabled: obsidianEnabledCheckbox.checked,
      obsidianVault: obsidianVaultInput.value.trim(),
      obsidianFolders: folders,
      obsidianLastFolder: lastFolder || folders[0]
    }, function () {
      saveBtn.textContent = 'Saved!';
      setTimeout(function () { saveBtn.textContent = 'Save'; }, 1500);
    });
  });
});
