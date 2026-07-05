const app = getApp()

function request(path, options = {}) {
  const { apiUrl, apiKey } = app.globalData
  return new Promise((resolve, reject) => {
    wx.request({
      url: apiUrl.replace(/\/$/, '') + path,
      method: options.method || 'GET',
      header: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer ' + apiKey
      },
      data: options.data,
      success(res) {
        if (res.statusCode >= 200 && res.statusCode < 300) {
          resolve(res.data)
        } else {
          reject(res.data)
        }
      },
      fail(err) {
        reject(err)
      }
    })
  })
}

function fetchArticles(limit = 20, offset = 0) {
  return request(`/api/v1/export?limit=${limit}&offset=${offset}`)
}

function deleteArticles(ids) {
  return request('/api/v1/delete', {
    method: 'POST',
    data: { ids }
  })
}

function saveArticle(url) {
  return request('/api/v1/save', {
    method: 'POST',
    data: { url }
  })
}

module.exports = { fetchArticles, deleteArticles, saveArticle }
