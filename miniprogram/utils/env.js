// 平台能力检测工具（小程序与多端 App 通用）。
// 判断当前运行环境是否具备「在系统浏览器打开外部链接」的能力：
//   多端 App 提供 wx.miniapp.openUrl，微信小程序没有该 API。

function canOpenExternalUrl() {
  try {
    return typeof wx !== 'undefined' && !!(wx.miniapp && typeof wx.miniapp.openUrl === 'function')
  } catch (e) {
    return false
  }
}

module.exports = { canOpenExternalUrl }
