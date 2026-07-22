const api = require('../../utils/api')
const app = getApp()

const BLOCK_TAGS = ['p', 'div', 'section', 'article', 'main']
const HEADING_TAGS = ['h1', 'h2', 'h3', 'h4', 'h5', 'h6']
const SKIP_TAGS = ['script', 'style', 'noscript', 'iframe', 'svg']

Page({
  data: {
    article: null,
    blocks: [],
    loading: true,
    error: ''
  },

  onLoad(options) {
    if (!app.checkSettings()) return
    const { id } = options
    if (!id) {
      this.setData({ loading: false, error: 'Missing article id' })
      return
    }
    this.loadArticle(id)
  },

  loadArticle(id) {
    this.setData({ loading: true, error: '' })
    api.fetchArticle(id).then(article => {
      const contentHtml = article.content_html || ''
      this.setData({
        article,
        blocks: parseArticleHtml(contentHtml, article.url),
        loading: false
      })
      wx.setNavigationBarTitle({ title: article.title || 'Reading' })
    }).catch(() => {
      this.setData({
        loading: false,
        error: 'Failed to load article'
      })
    })
  },

  onLinkTap(e) {
    const url = e.currentTarget.dataset.url
    if (!url) return
    wx.showActionSheet({
      itemList: ['Save to Read Later', 'Copy Link'],
      success: (res) => {
        if (res.tapIndex === 0) this.saveLink(url)
        if (res.tapIndex === 1) this.copyLink(url)
      }
    })
  },

  saveLink(url) {
    api.saveArticle(url).then(res => {
      wx.showToast({
        title: res && res.status === 'duplicate' ? 'Already saved' : 'Saved',
        icon: 'success'
      })
    }).catch(err => {
      wx.showToast({
        title: (err && err.error) || 'Failed to save',
        icon: 'none'
      })
    })
  },

  copyLink(url) {
    wx.setClipboardData({
      data: url,
      success() {
        wx.showToast({ title: 'Copied', icon: 'success' })
      }
    })
  },

  copyOriginalUrl() {
    const url = this.data.article && this.data.article.url
    if (!url) return
    this.copyLink(url)
  },

  confirmDelete() {
    wx.showModal({
      title: 'Delete article?',
      success: (res) => {
        if (res.confirm) this.deleteArticle()
      }
    })
  },

  deleteArticle() {
    const id = this.data.article && this.data.article.id
    if (!id) return
    api.deleteArticles([id]).then(() => {
      wx.showToast({ title: 'Deleted', icon: 'success' })
      setTimeout(() => wx.navigateBack(), 800)
    }).catch(err => {
      wx.showToast({
        title: (err && err.error) || 'Failed to delete',
        icon: 'none'
      })
    })
  }
})

function parseArticleHtml(html, baseUrl) {
  const blocks = []
  let block = null
  let linkUrl = ''
  const listStack = []
  let skipTag = ''
  let preDepth = 0

  function startBlock(type, extra = {}) {
    flushBlock()
    block = Object.assign({ type, segments: [] }, extra)
  }

  function ensureBlock() {
    if (!block) block = { type: 'p', segments: [] }
  }

  function addText(text) {
    if (!text) return
    const value = decodeEntities(text)
    if (!value || (!preDepth && !value.trim())) return
    ensureBlock()
    const normalized = preDepth ? value : value.replace(/\s+/g, ' ')
    const last = block.segments[block.segments.length - 1]
    const type = linkUrl ? 'link' : preDepth ? 'code' : 'text'
    if (last && last.type === type && last.url === linkUrl) {
      last.text += normalized
    } else {
      block.segments.push({ type, text: normalized, url: linkUrl })
    }
  }

  function addBreak() {
    ensureBlock()
    block.segments.push({ type: 'text', text: '\n', url: '' })
  }

  function flushBlock() {
    if (!block) return
    block.segments = block.segments.filter(seg => seg.text)
    const hasText = block.segments.some(seg => seg.type !== 'br' && seg.text && seg.text.trim())
    if (hasText || block.type === 'image') blocks.push(block)
    block = null
  }

  const tokens = String(html || '').match(/<!--[\s\S]*?-->|<\/?[^>]+>|[^<]+/g) || []
  tokens.forEach(token => {
    if (token.indexOf('<!--') === 0) return

    if (token[0] !== '<') {
      if (!skipTag) addText(token)
      return
    }

    const tagMatch = token.match(/^<\/?\s*([a-zA-Z0-9-]+)/)
    if (!tagMatch) return

    const tag = tagMatch[1].toLowerCase()
    const closing = /^<\//.test(token)
    const selfClosing = /\/\s*>$/.test(token)

    if (skipTag) {
      if (closing && tag === skipTag) skipTag = ''
      return
    }
    if (!closing && SKIP_TAGS.indexOf(tag) >= 0) {
      if (!selfClosing) skipTag = tag
      return
    }

    if (closing) {
      if (tag === 'a') linkUrl = ''
      if (tag === 'pre') {
        preDepth = Math.max(0, preDepth - 1)
        flushBlock()
      }
      if (tag === 'li' || tag === 'blockquote' || tag === 'p' || tag === 'div' || tag === 'section' || tag === 'article' || tag === 'main' || HEADING_TAGS.indexOf(tag) >= 0) {
        flushBlock()
      }
      if (tag === 'ul' || tag === 'ol') {
        const last = listStack[listStack.length - 1]
        if (last && last.type === tag) listStack.pop()
      }
      return
    }

    if (tag === 'a') {
      linkUrl = normalizeHref(getAttr(token, 'href'), baseUrl)
      return
    }
    if (tag === 'br') {
      addBreak()
      return
    }
    if (tag === 'img') {
      const src = normalizeHref(getAttr(token, 'src'), baseUrl)
      if (!src) return
      flushBlock()
      blocks.push({
        type: 'image',
        src,
        alt: decodeEntities(getAttr(token, 'alt') || '')
      })
      return
    }
    if (tag === 'ul' || tag === 'ol') {
      const start = Number(getAttr(token, 'start'))
      listStack.push({
        type: tag,
        count: tag === 'ol' && Number.isFinite(start) && start > 0 ? start - 1 : 0
      })
      flushBlock()
      return
    }
    if (tag === 'li') {
      const currentList = listStack[listStack.length - 1]
      let marker = '-'
      if (currentList && currentList.type === 'ol') {
        currentList.count += 1
        marker = currentList.count + '.'
      }
      startBlock('li', { ordered: !!(currentList && currentList.type === 'ol'), marker })
      return
    }
    if (tag === 'blockquote') {
      startBlock('quote')
      return
    }
    if (tag === 'pre') {
      preDepth += 1
      startBlock('pre')
      return
    }
    if (HEADING_TAGS.indexOf(tag) >= 0) {
      startBlock('heading', { level: Number(tag.slice(1)) })
      return
    }
    if (BLOCK_TAGS.indexOf(tag) >= 0) {
      flushBlock()
    }
  })
  flushBlock()
  return blocks
}

function getAttr(tag, name) {
  const pattern = new RegExp(name + "\\s*=\\s*(\"([^\"]*)\"|'([^']*)'|([^\\s>]+))", 'i')
  const match = tag.match(pattern)
  return match ? (match[2] || match[3] || match[4] || '') : ''
}

function decodeEntities(text) {
  return String(text || '')
    .replace(/&nbsp;/g, ' ')
    .replace(/&amp;/g, '&')
    .replace(/&lt;/g, '<')
    .replace(/&gt;/g, '>')
    .replace(/&quot;/g, '"')
    .replace(/&#39;/g, "'")
    .replace(/&#(\d+);/g, (_, code) => String.fromCharCode(Number(code)))
}

function normalizeHref(href, baseUrl) {
  const value = decodeEntities(href || '').trim()
  if (!value || value[0] === '#' || /^javascript:/i.test(value) || /^mailto:/i.test(value)) return ''
  if (/^https?:\/\//i.test(value)) return value
  if (value.indexOf('//') === 0) return 'https:' + value
  if (!baseUrl) return value

  const originMatch = baseUrl.match(/^(https?:\/\/[^/]+)/i)
  const origin = originMatch ? originMatch[1] : ''
  if (value[0] === '/') return origin + value

  const basePath = baseUrl.replace(/[?#].*$/, '').replace(/\/[^/]*$/, '/')
  return basePath + value
}
