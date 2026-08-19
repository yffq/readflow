// 系统分享 / 深链入口的工具函数（多端 App 与小程序通用）。
// 负责从启动参数中提取被分享的 URL、判断是否需要处理、生成跳转路径。
// 纯函数，便于单元测试。

function safeDecode(value) {
  try {
    return decodeURIComponent(value)
  } catch (e) {
    return value
  }
}

// 从一段文本里提炼出规范的 http(s) URL，提炼失败返回空串。
function normalize(raw) {
  if (!raw) return ''
  let s = String(raw).trim()
  // 去掉分享时可能夹带的引号、括号等包裹字符
  s = s.replace(/^["'[(\s]+/, '').replace(/["'\])\s]+$/, '')
  // 分享文本里可能混着描述 + URL，兜底提取第一个 http(s) 链接
  if (!/^https?:\/\//i.test(s)) {
    const m = s.match(/https?:\/\/[^\s"'<>]+/i)
    if (m) s = m[0]
  }
  // 去掉结尾可能混入的句子标点
  s = s.replace(/[),.;!?]+$/, '')
  return /^https?:\/\//i.test(s) ? s : ''
}

function extractFromObject(obj) {
  if (!obj || typeof obj !== 'object') return ''
  const keys = ['url', 'targetUrl', 'shareUrl', 'link', 'text']
  for (const key of keys) {
    if (obj[key]) {
      const value = normalize(String(obj[key]))
      if (value) return value
    }
  }
  return ''
}

function extractFromQuery(query) {
  if (typeof query === 'string') {
    // 形如 "url=xxx&a=b" 或直接是裸 URL
    const m = query.match(/(?:^|[?&])url=([^&]+)/i)
    if (m) return normalize(safeDecode(m[1]))
    return normalize(query)
  }
  return extractFromObject(query)
}

function extractFromExtra(extra) {
  if (typeof extra === 'string') return normalize(extra)
  return extractFromObject(extra)
}

function extractFromPath(path) {
  if (typeof path !== 'string') return ''
  // 深链可能是 "pages/save/save?url=xxx" 或直接是 URL
  const m = path.match(/(?:^|[?&])url=([^&]+)/i)
  if (m) return normalize(safeDecode(m[1]))
  return normalize(path)
}

// 从启动参数里提取被分享的 URL，提取不到返回空串。
function extractShareUrl(options) {
  if (!options) return ''

  const fromQuery = extractFromQuery(options.query)
  if (fromQuery) return fromQuery

  const referrerInfo = options.referrerInfo || {}
  const fromExtra = extractFromExtra(referrerInfo.extraData)
  if (fromExtra) return fromExtra

  return extractFromPath(options.path)
}

// 是否需要处理本次分享：有 URL 且与上次已处理的 URL 不同。
function shouldHandleShare(url, lastHandledUrl) {
  return !!url && url !== lastHandledUrl
}

// 生成跳转到保存页的路径。
function buildSavePath(url) {
  return '/pages/save/save?url=' + encodeURIComponent(url)
}

// 根据是否已配置服务端信息，决定对本次分享采取的动作。
// 返回 null（无需处理）或 { action: 'save'|'settings', url, path }。
function planShare(url, hasSettings) {
  if (!url) return null
  if (hasSettings) {
    return { action: 'save', url, path: buildSavePath(url) }
  }
  return { action: 'settings', url, path: '/pages/settings/settings' }
}

module.exports = {
  extractShareUrl,
  shouldHandleShare,
  buildSavePath,
  planShare
}
