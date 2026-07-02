Page({
  data: {
    id: '',
    title: '',
    author: '',
    sitename: '',
    htmlContent: ''
  },

  onLoad(options) {
    const { id, title, author, sitename } = options
    let html = wx.getStorageSync('article_' + id) || ''
    html = html
      .replace(/<img /g, '<img style="max-width:100%;height:auto" ')
      .replace(/<table/g, '<table style="display:block;max-width:100%;overflow-x:auto" ')
      .replace(/<pre/g, '<pre style="white-space:pre-wrap;word-break:break-all" ')
    this.setData({
      id,
      title: decodeURIComponent(title || ''),
      author: decodeURIComponent(author || ''),
      sitename: decodeURIComponent(sitename || ''),
      htmlContent: html
    })
  }
})
