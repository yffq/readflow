const app = getApp()

function request(path, options = {}) {
  const { apiUrl, apiKey } = app.globalData
  return new Promise((resolve, reject) => {
    if (!apiUrl || !apiKey) {
      reject({ error: 'API URL or API key is missing' })
      return
    }
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
          reject(Object.assign({ statusCode: res.statusCode }, res.data || {}))
        }
      },
      fail(err) {
        reject(err)
      }
    })
  })
}

function fetchArticles(limit = 20, offset = 0, asc = false) {
  var params = 'limit=' + limit + '&offset=' + offset + '&content=false&count=false';
  if (asc) params += '&sort=asc';
  return request('/api/v1/export?' + params)
}

function fetchArticleSummaries(limit = 20, offset = 0, asc = false) {
  var params = 'limit=' + limit + '&offset=' + offset + '&content=false';
  if (asc) params += '&sort=asc';
  return request('/api/v1/export?' + params)
}

function fetchArticle(id) {
  return request('/api/v1/article/' + encodeURIComponent(id))
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

module.exports = { fetchArticles, fetchArticleSummaries, fetchArticle, deleteArticles, saveArticle }
