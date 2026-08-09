// Unit and end-to-end tests for opening the Obsidian protocol URL.
// Run: node extension/obsidian-navigation-test.js

var navigation = require('./obsidian-navigation.js');
var assert = require('assert');

async function run() {
  var clipboardUri = navigation.buildNewNoteUri('My Vault', 'Readflow/Test Note.md', 'full body', true);
  assert(clipboardUri.indexOf('file=Readflow%2FTest%20Note.md') !== -1);
  assert(clipboardUri.indexOf('vault=My%20Vault') !== -1);
  assert(clipboardUri.indexOf('&clipboard&') !== -1);
  assert(clipboardUri.indexOf(encodeURIComponent('full body')) === -1);

  var fallbackUri = navigation.buildNewNoteUri('', 'Readflow/Test.md', 'full body', false);
  assert(fallbackUri.indexOf('clipboard') === -1);
  assert(fallbackUri.indexOf('content=full%20body') !== -1);

  var calls = [];
  var chromeApi = {
    tabs: {
      query: async function (query) {
        calls.push({ method: 'query', args: query });
        return [{ id: 42, url: 'https://example.com/article' }];
      },
      update: async function (id, update) {
        calls.push({ method: 'update', id: id, args: update });
      }
    }
  };

  await navigation.openInCurrentTab(chromeApi, 'obsidian://new?file=Readflow/Test.md');
  assert.deepStrictEqual(calls[0], { method: 'query', args: { active: true, currentWindow: true } });
  assert.deepStrictEqual(calls[1], {
    method: 'update',
    id: 42,
    args: { url: 'obsidian://new?file=Readflow/Test.md' }
  });
  assert.strictEqual(calls.some(function (call) { return call.method === 'create'; }), false);

  await assert.rejects(function () {
    return navigation.openInCurrentTab({ tabs: { query: async function () { return []; } } }, 'obsidian://new?file=x');
  }, /No active browser tab/);
  await assert.rejects(function () {
    return navigation.openInCurrentTab(chromeApi, 'https://example.com');
  }, /Invalid Obsidian URL/);

  var failedChrome = {
    tabs: {
      query: async function () { return [{ id: 7 }]; },
      update: async function () { throw new Error('Protocol handler rejected'); }
    }
  };
  await assert.rejects(function () {
    return navigation.openInCurrentTab(failedChrome, 'obsidian://new?file=x');
  }, /Protocol handler rejected/);

  console.log('Obsidian navigation tests passed');
}

run().catch(function (err) {
  console.error(err);
  process.exit(1);
});
