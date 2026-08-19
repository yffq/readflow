const app = getApp()
const { buildSavePath } = require('../../utils/share')

Page({
  data: {
    apiUrl: '',
    apiKey: ''
  },

  onLoad() {
    this.setData({
      apiUrl: app.globalData.apiUrl,
      apiKey: app.globalData.apiKey
    })
  },

  onApiUrlChange(e) {
    this.setData({ apiUrl: e.detail.value })
  },

  onApiKeyChange(e) {
    this.setData({ apiKey: e.detail.value })
  },

  saveSettings() {
    const { apiUrl, apiKey } = this.data
    if (!apiUrl || !apiKey) {
      wx.showToast({ title: 'Fill all fields', icon: 'none' })
      return
    }
    wx.setStorageSync('settings', { apiUrl, apiKey })
    app.globalData.apiUrl = apiUrl
    app.globalData.apiKey = apiKey
    wx.showToast({ title: 'Saved', icon: 'success' })
    setTimeout(() => {
      const pending = app.globalData.pendingShareUrl
      if (pending) {
        // 设置前有未处理的分享 URL，设置完成后直接进入保存页
        app.globalData.pendingShareUrl = ''
        wx.redirectTo({ url: buildSavePath(pending) })
      } else {
        wx.redirectTo({ url: '/pages/index/index' })
      }
    }, 1000)
  }
})
