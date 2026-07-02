const api = require('../../utils/api')
const app = getApp()

Page({
  data: {
    articles: [],
    total: 0,
    page: 1,
    limit: 20,
    loading: true
  },

  onShow() {
    if (!app.checkSettings()) return
    this.loadArticles()
  },

  onPullDownRefresh() {
    this.setData({ page: 1 })
    this.loadArticles(true).finally(() => wx.stopPullDownRefresh())
  },

  loadArticles(showLoading) {
    if (showLoading !== false && this.data.articles.length === 0) {
      this.setData({ loading: true })
    }
    const offset = (this.data.page - 1) * this.data.limit
    return api.fetchArticles(this.data.limit, offset).then(res => {
      this.setData({
        articles: res.results || [],
        total: res.count || 0,
        loading: false
      })
    }).catch(err => {
      this.setData({ loading: false })
      if (showLoading !== false) {
        wx.showToast({ title: 'Failed to load', icon: 'none' })
      }
    })
  },

  openArticle(e) {
    const { id, title, html, url, author, sitename } = e.currentTarget.dataset
    wx.setStorageSync('article_' + id, html || '')
    wx.navigateTo({
      url: `/pages/read/read?id=${id}&title=${encodeURIComponent(title)}&url=${encodeURIComponent(url || '')}&author=${encodeURIComponent(author || '')}&sitename=${encodeURIComponent(sitename || '')}`
    })
  },

  confirmDelete(e) {
    const id = e.currentTarget.dataset.id
    wx.showModal({
      title: 'Delete article?',
      success: (res) => {
        if (res.confirm) this.deleteArticle(id)
      }
    })
  },

  deleteArticle(id) {
    api.deleteArticles([id]).then(() => {
      const articles = this.data.articles.filter(a => a.id !== id)
      this.setData({ articles, total: this.data.total - 1 })
      wx.showToast({ title: 'Deleted', icon: 'success' })
    }).catch(() => {
      wx.showToast({ title: 'Failed', icon: 'none' })
    })
  },

  prevPage() {
    if (this.data.page <= 1) return
    this.setData({ page: this.data.page - 1 }, () => this.loadArticles())
  },

  nextPage() {
    if (this.data.total <= this.data.page * this.data.limit) return
    this.setData({ page: this.data.page + 1 }, () => this.loadArticles())
  },

  openSettings() {
    wx.navigateTo({ url: '/pages/settings/settings' })
  }
})
