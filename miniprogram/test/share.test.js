const test = require('node:test')
const assert = require('node:assert')
const {
  extractShareUrl,
  shouldHandleShare,
  buildSavePath,
  planShare
} = require('../utils/share')

test('extractShareUrl: query.url 对象形式', () => {
  assert.strictEqual(
    extractShareUrl({ query: { url: 'https://example.com/a' } }),
    'https://example.com/a'
  )
})

test('extractShareUrl: query 字符串形式带 url 参数', () => {
  assert.strictEqual(
    extractShareUrl({ query: 'url=https%3A%2F%2Fexample.com%2Fa&x=1' }),
    'https://example.com/a'
  )
})

test('extractShareUrl: query 字符串形式是裸 URL', () => {
  assert.strictEqual(
    extractShareUrl({ query: 'https://example.com/b?x=1' }),
    'https://example.com/b?x=1'
  )
})

test('extractShareUrl: referrerInfo.extraData.url', () => {
  assert.strictEqual(
    extractShareUrl({ referrerInfo: { extraData: { url: 'https://example.com/c' } } }),
    'https://example.com/c'
  )
})

test('extractShareUrl: path 里带 url 参数', () => {
  assert.strictEqual(
    extractShareUrl({ path: 'pages/save/save?url=https%3A%2F%2Fexample.com%2Fd' }),
    'https://example.com/d'
  )
})

test('extractShareUrl: 分享文本里夹带 URL 时提取第一个链接', () => {
  assert.strictEqual(
    extractShareUrl({ query: { text: '看看这篇 https://example.com/e 很棒' } }),
    'https://example.com/e'
  )
})

test('extractShareUrl: 空参数返回空串', () => {
  assert.strictEqual(extractShareUrl(undefined), '')
  assert.strictEqual(extractShareUrl(null), '')
  assert.strictEqual(extractShareUrl({}), '')
  assert.strictEqual(extractShareUrl({ query: {} }), '')
})

test('extractShareUrl: 非 http(s) 内容被拒绝', () => {
  assert.strictEqual(extractShareUrl({ query: { url: 'javascript:alert(1)' } }), '')
  assert.strictEqual(extractShareUrl({ query: 'ftp://example.com/f' }), '')
})

test('extractShareUrl: 去掉结尾标点和包裹字符', () => {
  assert.strictEqual(
    extractShareUrl({ query: { url: '"https://example.com/g."' } }),
    'https://example.com/g'
  )
})

test('shouldHandleShare: 有 URL 且与上次不同才处理', () => {
  assert.strictEqual(shouldHandleShare('https://a.com', ''), true)
  assert.strictEqual(shouldHandleShare('https://a.com', 'https://a.com'), false)
  assert.strictEqual(shouldHandleShare('', ''), false)
})

test('buildSavePath: 对 URL 进行编码', () => {
  const url = 'https://example.com/a?x=1&y=2'
  assert.strictEqual(buildSavePath(url), '/pages/save/save?url=' + encodeURIComponent(url))
})

test('planShare: 已配置走 save，未配置走 settings，无 URL 返回 null', () => {
  assert.deepStrictEqual(planShare('https://a.com', true), {
    action: 'save',
    url: 'https://a.com',
    path: '/pages/save/save?url=' + encodeURIComponent('https://a.com')
  })
  assert.deepStrictEqual(planShare('https://a.com', false), {
    action: 'settings',
    url: 'https://a.com',
    path: '/pages/settings/settings'
  })
  assert.strictEqual(planShare('', true), null)
  assert.strictEqual(planShare(undefined, false), null)
})
