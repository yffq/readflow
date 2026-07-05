const api = require('../../utils/api')

function stripTags(html) {
  return html.replace(/<[^>]+>/g, '')
}

function parseInline(html) {
  const parts = []
  const re = /<a\b[^>]*href="([^"]*)"[^>]*>([\s\S]*?)<\/a>|<strong\b[^>]*>([\s\S]*?)<\/strong>|<em\b[^>]*>([\s\S]*?)<\/em>|<code\b[^>]*>([\s\S]*?)<\/code>/gi
  let last = 0
  let m
  while ((m = re.exec(html)) !== null) {
    if (m.index > last) {
      const t = stripTags(html.slice(last, m.index))
      if (t) parts.push({ type: 'text', content: t })
    }
    if (m[1]) {
      parts.push({ type: 'link', href: m[1], text: stripTags(m[2]) })
    } else if (m[3] !== undefined) {
      parts.push({ type: 'strong', content: stripTags(m[3]) })
    } else if (m[4] !== undefined) {
      parts.push({ type: 'em', content: stripTags(m[4]) })
    } else if (m[5] !== undefined) {
      parts.push({ type: 'code', content: stripTags(m[5]) })
    }
    last = m.index + m[0].length
  }
  if (last < html.length) {
    const t = stripTags(html.slice(last))
    if (t) parts.push({ type: 'text', content: t })
  }
  if (parts.length === 0) {
    const t = stripTags(html)
    if (t) parts.push({ type: 'text', content: t })
  }
  return parts
}

function parseHTML(html) {
  const nodes = []
  html = html
    .replace(/<style[^>]*>[\s\S]*?<\/style>/gi, '')
    .replace(/<script[^>]*>[\s\S]*?<\/script>/gi, '')
    .replace(/<figcaption[^>]*>[\s\S]*?<\/figcaption>/gi, '')
    .replace(/<\/?figure[^>]*>/gi, '')

  const blockRe = /<(h[1-6]|p|pre|img|blockquote|li|ul|ol)\b[^>]*>/gi
  const blocks = html.split(/(<\/?(?:h[1-6]|p|pre|img|blockquote|li|ul|ol)\b[^>]*>)/gi)

  let listItems = []
  let inList = false

  function flushList() {
    if (listItems.length > 0) {
      nodes.push({ type: 'list', items: listItems })
      listItems = []
      inList = false
    }
  }

  let i = 0
  while (i < blocks.length) {
    const s = blocks[i] || ''
    if (/^<h[1-6]/.test(s)) {
      flushList()
      const level = parseInt(s.charAt(2))
      const content = blocks[i + 1] || ''
      const text = stripTags(content).trim()
      if (text) nodes.push({ type: 'heading', level, content: text })
      i += 2
      continue
    }
    if (/^<p/.test(s)) {
      flushList()
      const content = blocks[i + 1] || ''
      const parts = parseInline(content)
      if (parts.length > 0) nodes.push({ type: 'para', parts })
      i += 2
      continue
    }
    if (/^<pre/.test(s)) {
      flushList()
      const content = blocks[i + 1] || ''
      const text = stripTags(content)
      if (text.trim()) nodes.push({ type: 'code', content: text })
      i += 2
      continue
    }
    if (/^<blockquote/.test(s)) {
      flushList()
      const content = blocks[i + 1] || ''
      const parts = parseInline(content)
      if (parts.length > 0) nodes.push({ type: 'quote', parts })
      i += 2
      continue
    }
    if (/^<img/.test(s)) {
      flushList()
      const src = (s.match(/src="([^"]*)"/) || [])[1]
      if (src) nodes.push({ type: 'image', src })
      i++
      continue
    }
    if (/^<li/.test(s)) {
      inList = true
      const content = blocks[i + 1] || ''
      const parts = parseInline(content)
      if (parts.length > 0) listItems.push(parts)
      i += 2
      continue
    }
    if (/^<\/ul>|^<\/ol>/.test(s)) {
      flushList()
      i++
      continue
    }
    if (/^<ul|^<ol/.test(s)) {
      i++
      continue
    }
    if (/^<\//.test(s)) {
      i++
      continue
    }
    const text = stripTags(s).trim()
    if (text) {
      flushList()
      nodes.push({ type: 'para', parts: [{ type: 'text', content: text }] })
    }
    i++
  }
  flushList()
  return nodes
}

Page({
  data: {
    id: '',
    title: '',
    author: '',
    sitename: '',
    url: '',
    nodes: [],
    loading: true
  },

  onLoad(options) {
    const { id, title, author, sitename, url } = options
    const html = wx.getStorageSync('article_' + id) || ''
    const nodes = parseHTML(html)

    this.setData({
      id,
      title: decodeURIComponent(title || ''),
      author: decodeURIComponent(author || ''),
      sitename: decodeURIComponent(sitename || ''),
      url: decodeURIComponent(url || ''),
      nodes,
      loading: false
    })
  },

  openOriginal() {
    const url = this.data.url
    if (url) {
      wx.setClipboardData({ data: url, success: () => wx.showToast({ title: '链接已复制', icon: 'success' }) })
    }
  },

  saveLink(e) {
    const href = e.currentTarget.dataset.url
    if (!href) return
    wx.showActionSheet({
      itemList: ['保存到 Read Later', '复制链接'],
      success: (res) => {
        if (res.tapIndex === 0) {
          api.saveArticle(href).then(() => {
            wx.showToast({ title: '已保存', icon: 'success' })
          }).catch(() => {
            wx.showToast({ title: '保存失败', icon: 'none' })
          })
        } else {
          wx.setClipboardData({ data: href, success: () => wx.showToast({ title: '已复制', icon: 'success' }) })
        }
      }
    })
  }
})
