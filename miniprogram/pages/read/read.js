Page({
  data: {
    id: '',
    title: '',
    author: '',
    sitename: '',
    url: '',
    htmlContent: '',
    loading: true
  },

  onLoad(options) {
    const { id, title, author, sitename, url } = options
    let html = wx.getStorageSync('article_' + id) || ''

    html = html
      .replace(/<img /g, '<img style="max-width:100%;height:auto" ')
      .replace(/<table/g, '<table style="display:block;max-width:100%;overflow-x:auto" ')
      .replace(/<pre/g, '<pre style="white-space:pre-wrap;word-break:break-all" ')
      .replace(/<a [^>]*href="([^"]*)"[^>]*>([^<]*)<\/a>/g, '$2 ($1)')

    this.setData({
      id,
      title: decodeURIComponent(title || ''),
      author: decodeURIComponent(author || ''),
      sitename: decodeURIComponent(sitename || ''),
      url: decodeURIComponent(url || ''),
      htmlContent: html,
      loading: false
    })
  },

  openOriginal() {
    const url = this.data.url
    if (url) {
      wx.setClipboardData({ data: url, success: () => wx.showToast({ title: '链接已复制', icon: 'success' }) })
    }
  }
})
