const app = getApp()

Page({
  data: { url: '' },

  onLoad(options) {
    const { id } = options
    const apiKey = app.globalData.apiKey
    const serverUrl = app.globalData.apiUrl.replace(/\/$/, '')
    const url = serverUrl + '/api/v1/read/' + id + '?api_key=' + encodeURIComponent(apiKey)
    this.setData({ url })
  }
})
