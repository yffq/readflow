const api = require('../../utils/api')
const app = getApp()

Page({
  data: {
    url: '',
    saving: false,
    error: ''
  },

  onLoad(options) {
    if (!app.checkSettings()) return
    if (options && options.url) {
      this.setData({ url: decodeURIComponent(options.url) })
    }
  },

  onUrlChange(e) {
    this.setData({ url: e.detail.value, error: '' })
  },

  pasteFromClipboard() {
    wx.getClipboardData({
      success: (res) => {
        this.setData({ url: (res.data || '').trim(), error: '' })
      },
      fail: () => {
        this.showError('Failed to read clipboard')
      }
    })
  },

  saveArticle() {
    const url = this.data.url.trim()
    if (!url) {
      this.showError('URL is required')
      return
    }
    if (!/^https?:\/\//i.test(url)) {
      this.showError('URL must start with http:// or https://')
      return
    }
    if (this.data.saving) return

    this.setData({ saving: true, error: '' })
    api.saveArticle(url).then(res => {
      const title = res && res.status === 'duplicate' ? 'Already saved' : 'Saved'
      wx.showToast({ title, icon: 'success' })
      setTimeout(() => {
        const pages = getCurrentPages()
        if (pages.length > 1) {
          wx.navigateBack()
        } else {
          wx.redirectTo({ url: '/pages/index/index' })
        }
      }, 600)
    }).catch(err => {
      this.showError(this.errorMessage(err))
    }).finally(() => {
      this.setData({ saving: false })
    })
  },

  showError(message) {
    this.setData({ error: message })
    wx.showToast({ title: message, icon: 'none', duration: 3000 })
  },

  errorMessage(err) {
    if (!err) return 'Failed to save'
    if (err.error) return err.error
    if (err.errMsg) return err.errMsg
    if (err.statusCode) return 'Request failed: ' + err.statusCode
    return 'Failed to save'
  }
})
