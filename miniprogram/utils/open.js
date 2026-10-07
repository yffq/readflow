const { canOpenExternalUrl } = require('./env')

// 打开外部链接：多端 App 调系统浏览器；小程序（能力缺失）或非法 URL 返回 false，由调用方兜底。
// 返回 true 表示已把打开动作交给 openUrl（异步，成功与否以 fail 回调为准）。
function openExternalUrl(url) {
  if (!url || !/^https?:\/\//i.test(url)) return false
  if (!canOpenExternalUrl()) return false

  wx.miniapp.openUrl({
    url,
    fail() {
      // 打开失败时给用户明确提示，并提供复制链接兜底
      wx.showModal({
        title: 'Cannot open',
        content: 'Failed to open in browser. Copy the link instead?',
        success: (res) => {
          if (res.confirm) wx.setClipboardData({ data: url })
        }
      })
    }
  })
  return true
}

module.exports = { openExternalUrl }
