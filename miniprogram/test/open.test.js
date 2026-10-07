const test = require('node:test')
const assert = require('node:assert')
const { canOpenExternalUrl } = require('../utils/env')
const { openExternalUrl } = require('../utils/open')

test('canOpenExternalUrl: wx 未定义时返回 false', () => {
  delete global.wx
  assert.strictEqual(canOpenExternalUrl(), false)
})

test('canOpenExternalUrl: 有 wx 但无 miniapp 时返回 false（小程序环境）', () => {
  global.wx = {}
  assert.strictEqual(canOpenExternalUrl(), false)
  delete global.wx
})

test('canOpenExternalUrl: miniapp.openUrl 为函数时返回 true（App 环境）', () => {
  global.wx = { miniapp: { openUrl: () => {} } }
  assert.strictEqual(canOpenExternalUrl(), true)
  delete global.wx
})

test('canOpenExternalUrl: miniapp.openUrl 非函数时返回 false', () => {
  global.wx = { miniapp: { openUrl: 'not-a-function' } }
  assert.strictEqual(canOpenExternalUrl(), false)
  delete global.wx
})

test('openExternalUrl: 空值或非 http(s) scheme 返回 false', () => {
  global.wx = { miniapp: { openUrl: () => {} } }
  assert.strictEqual(openExternalUrl(''), false)
  assert.strictEqual(openExternalUrl(null), false)
  assert.strictEqual(openExternalUrl(undefined), false)
  assert.strictEqual(openExternalUrl('javascript:alert(1)'), false)
  assert.strictEqual(openExternalUrl('ftp://example.com/f'), false)
  assert.strictEqual(openExternalUrl('weixin://dl/moments'), false)
  delete global.wx
})

test('openExternalUrl: 小程序环境（无 openUrl）返回 false', () => {
  global.wx = {}
  assert.strictEqual(openExternalUrl('https://example.com/a'), false)
  delete global.wx
})

test('openExternalUrl: App 环境调用 openUrl 并返回 true（https 与 http）', () => {
  const calls = []
  global.wx = { miniapp: { openUrl: (opts) => calls.push(opts.url) } }
  assert.strictEqual(openExternalUrl('https://example.com/a'), true)
  assert.strictEqual(openExternalUrl('http://example.com/b'), true)
  assert.deepStrictEqual(calls, ['https://example.com/a', 'http://example.com/b'])
  delete global.wx
})

test('openExternalUrl: openUrl fail 时弹窗，确认后复制链接', () => {
  let failCb = null
  let modalCb = null
  let copied = ''
  global.wx = {
    miniapp: { openUrl: (opts) => { failCb = opts.fail } },
    showModal: (opts) => { modalCb = opts.success },
    setClipboardData: (opts) => { copied = opts.data }
  }
  assert.strictEqual(openExternalUrl('https://example.com/a'), true)

  assert.strictEqual(typeof failCb, 'function')
  failCb()

  assert.strictEqual(typeof modalCb, 'function')
  modalCb({ confirm: true })
  assert.strictEqual(copied, 'https://example.com/a')
  delete global.wx
})

test('openExternalUrl: openUrl fail 后用户取消则不复制', () => {
  let failCb = null
  let modalCb = null
  let copied = ''
  global.wx = {
    miniapp: { openUrl: (opts) => { failCb = opts.fail } },
    showModal: (opts) => { modalCb = opts.success },
    setClipboardData: (opts) => { copied = opts.data }
  }
  openExternalUrl('https://example.com/a')
  failCb()
  modalCb({ confirm: false })
  assert.strictEqual(copied, '')
  delete global.wx
})
