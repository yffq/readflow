App({
  globalData: {
    apiUrl: '',
    apiKey: ''
  },

  onLaunch() {
    const settings = wx.getStorageSync('settings')
    if (settings) {
      this.globalData.apiUrl = settings.apiUrl || ''
      this.globalData.apiKey = settings.apiKey || ''
    }
  },

  checkSettings() {
    if (!this.globalData.apiUrl || !this.globalData.apiKey) {
      wx.redirectTo({ url: '/pages/settings/settings' })
      return false
    }
    return true
  }
})
