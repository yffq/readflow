const { extractShareUrl, shouldHandleShare, planShare } = require('./utils/share')

App({
  globalData: {
    apiUrl: '',
    apiKey: '',
    pendingShareUrl: ''
  },

  _lastShareUrl: '',

  onLaunch(options) {
    const settings = wx.getStorageSync('settings')
    if (settings) {
      this.globalData.apiUrl = settings.apiUrl || ''
      this.globalData.apiKey = settings.apiKey || ''
    }
    // 冷启动时 onLaunch 之后一定会触发 onShow，分享入口统一交给 onShow 处理。
  },

  onShow(options) {
    const url = extractShareUrl(options)
    if (shouldHandleShare(url, this._lastShareUrl)) {
      this._lastShareUrl = url
      this.queueShare(url)
    }
  },

  onHide() {
    // 下次回到前台时允许处理一份新的分享内容
    this._lastShareUrl = ''
  },

  queueShare(url) {
    const hasSettings = !!(this.globalData.apiUrl && this.globalData.apiKey)
    const plan = planShare(url, hasSettings)
    if (!plan) return

    if (plan.action === 'settings') {
      // 尚未配置服务端信息，先记录待分享 URL，引导用户去设置
      this.globalData.pendingShareUrl = plan.url
      wx.redirectTo({ url: plan.path })
      return
    }

    this.globalData.pendingShareUrl = ''
    wx.navigateTo({ url: plan.path })
  },

  checkSettings() {
    if (!this.globalData.apiUrl || !this.globalData.apiKey) {
      wx.redirectTo({ url: '/pages/settings/settings' })
      return false
    }
    return true
  }
})
