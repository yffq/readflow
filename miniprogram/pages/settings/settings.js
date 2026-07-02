const app = getApp()

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
    setTimeout(() => wx.redirectTo({ url: '/pages/index/index' }), 1000)
  }
})
