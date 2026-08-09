// Unit and end-to-end tests for the Readflow save pipeline.
// Run: node extension/save-test.js

var api = require('./readflow-api.js');
var assert = require('assert');

async function run() {
  assert.throws(function () { api.validateSettings({ apiKey: 'rf_test', serverUrl: '' }); }, /Server URL/);
  assert.throws(function () { api.validateSettings({ apiKey: '', serverUrl: 'https://example.com' }); }, /API key/);
  assert.throws(function () { api.validateSettings({ apiKey: 'rf_test', serverUrl: 'example.com' }); }, /invalid/);
  assert.throws(function () { api.validatePageUrl('chrome://settings'); }, /Only http/);
  assert.strictEqual(api.validatePageUrl('https://example.com/a'), 'https://example.com/a');

  var captured;
  var result = await api.savePage(
    { serverUrl: 'https://readflow.example/', apiKey: 'rf_test' },
    { url: 'https://example.com/article', title: 'Article' },
    async function (url, options) {
      captured = { url: url, options: options };
      return { ok: true, status: 200, text: async function () { return '{"id":"1","status":"created"}'; } };
    }
  );
  assert.strictEqual(captured.url, 'https://readflow.example/api/v1/save');
  assert.strictEqual(captured.options.headers.Authorization, 'Bearer rf_test');
  assert.deepStrictEqual(JSON.parse(captured.options.body), { url: 'https://example.com/article', title: 'Article' });
  assert.strictEqual(result.status, 'created');

  await assert.rejects(function () {
    return api.savePage(
      { serverUrl: 'https://readflow.example', apiKey: 'bad' },
      { url: 'https://example.com' },
      async function () { return { ok: false, status: 401, text: async function () { return '{"error":"invalid api key"}'; } }; }
    );
  }, /invalid api key/);

  await assert.rejects(function () {
    return api.savePage(
      { serverUrl: 'https://readflow.example', apiKey: 'rf_test' },
      { url: 'https://example.com' },
      async function () { throw new Error('offline'); }
    );
  }, /Cannot reach/);

  console.log('Readflow save tests passed');
}

run().catch(function (err) {
  console.error(err);
  process.exit(1);
});
