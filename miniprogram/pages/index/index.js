const api = require('../../utils/api')
const app = getApp()

Page({
  data: {
    articles: [],
    total: 0,
    page: 1,
    limit: 20,
    loading: true,
    refreshing: false,
    hasMore: false,
    error: ''
  },

  onShow() {
    if (!app.checkSettings()) return
    this.loadArticles()
  },

  onPullDownRefresh() {
    if (this.data.refreshing) {
      wx.stopPullDownRefresh()
      return
    }
    wx.showNavigationBarLoading()
    this.setData({ page: 1, refreshing: true })
    this.loadArticles(false).finally(() => {
      this.setData({ refreshing: false })
      wx.hideNavigationBarLoading()
      wx.stopPullDownRefresh()
    })
  },

  loadArticles(showLoading) {
    if (showLoading !== false && this.data.articles.length === 0) {
      this.setData({ loading: true, error: '' })
    } else {
      this.setData({ error: '' })
    }
    const offset = (this.data.page - 1) * this.data.limit
    return this.fetchArticlePage(this.data.limit, offset).then(res => {
      const months = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']
      const articles = (res.results || []).map(a => {
        const d = a.saved_at ? new Date(a.saved_at) : null
        const saved_at = d && !isNaN(d.getTime()) ? months[d.getMonth()] + ' ' + String(d.getDate()).padStart(2, '0') + ', ' + d.getFullYear() : a.saved_at
        return Object.assign({}, a, { saved_at })
      })
      this.setData({
        articles,
        total: res.count || 0,
        hasMore: !!(res.has_more || res.next),
        loading: false,
        error: ''
      })
    }).catch(err => {
      const msg = this.errorMessage(err)
      this.setData({ loading: false, error: msg })
      wx.showToast({ title: msg, icon: 'none', duration: 3000 })
    })
  },

  fetchArticlePage(limit, offset) {
    return api.fetchArticles(limit, offset).catch(err => {
      return api.fetchArticleSummaries(limit, offset).catch(() => {
        throw err
      })
    })
  },

  errorMessage(err) {
    if (!err) return 'Failed to load'
    if (err.error) return err.error
    if (err.errMsg) return err.errMsg
    if (err.statusCode) return 'Request failed: ' + err.statusCode
    return 'Failed to load'
  },

  openArticle(e) {
    const { id } = e.currentTarget.dataset
    wx.navigateTo({ url: '/pages/read/read?id=' + id })
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
    const previousArticles = this.data.articles
    const previousTotal = this.data.total
    const articles = previousArticles.filter(a => a.id !== id)
    this.setData({ articles, total: Math.max(0, previousTotal - 1) })

    api.deleteArticles([id]).then(() => {
      wx.showToast({ title: 'Deleted', icon: 'success' })
    }).catch(() => {
      this.setData({ articles: previousArticles, total: previousTotal })
      wx.showToast({ title: 'Failed', icon: 'none' })
    })
  },

  prevPage() {
    if (this.data.page <= 1) return
    this.setData({ page: this.data.page - 1 }, () => this.loadArticles())
  },

  nextPage() {
    if (!this.data.hasMore) return
    this.setData({ page: this.data.page + 1 }, () => this.loadArticles())
  },

  openSave() {
    wx.navigateTo({ url: '/pages/save/save' })
  },

  openSettings() {
    wx.navigateTo({ url: '/pages/settings/settings' })
  }
})
